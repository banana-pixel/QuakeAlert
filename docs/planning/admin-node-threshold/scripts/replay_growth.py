#!/usr/bin/env python3
# research-only — offline QuakeAlert-analog replay of the intra-event PGA growth curve
# (task 7c8d / doc 16 §§6-11). NOT production code. Read-only: fetches public FDSN
# waveforms, writes only local result files under scripts/out/. Nothing is sent back.
#
# Faithfully mirrors firmware/src/sensor.cpp + config.h (verified 2026-09-27):
#   baselineEMA = a*mag + (1-a)*baselineEMA           a=BASELINE_ALPHA=0.005
#   corrected   = |mag - baselineEMA|
#   sta = STA_ALPHA*corr + (1-STA_ALPHA)*sta          STA_ALPHA=1/50
#   lta = LTA_ALPHA*corr + (1-LTA_ALPHA)*lta (FROZEN while eventInProgress)  LTA_ALPHA=1/2000
#   ratio = sta / max(lta, 0.5);  trigger ratio>=4.0 & sta>=1.5; confirm 300 ms; detrigger ratio<1.5
#   PRELIM = corrected at confirmation instant (onset sample);  running peak from confirmation on
# Non-equivalence vs the real device is DOCUMENTED in doc 16 §Critical Requirement, not hidden.
import sys, json, math, os
import numpy as np
from obspy import UTCDateTime
from obspy.clients.fdsn import Client
from obspy.geodetics import gps2dist_azimuth

# ---- exact firmware constants (config.h) ----
BASELINE_ALPHA = 0.005
STA_ALPHA      = 1.0/50.0
LTA_ALPHA      = 1.0/2000.0
MIN_LTA_CLAMP  = 0.5
TRIG_RATIO     = 4.0
DETRIG_RATIO   = 1.5
MIN_STA_GAL    = 1.5
CONFIRM_S      = 0.300
MAX_EVENT_S    = 60.0
SR             = 100.0            # device assumed rate; strong-motion HN? are 100 sps
CHECKPOINTS    = [0.3, 0.5, 1.0, 1.5, 2.0, 3.0]   # s after T0 (confirmation)
THRESHOLDS     = [60, 80, 100, 120, 140]          # gal — RESEARCH COMPARISON ONLY
WARMUP_S       = 30.0            # analog of LTA warm-up: let LTA settle on pre-event noise

# well-documented, FDSN-open events (CI network, HN? = 100 sps strong motion)
EVENTS = [
    dict(name="Ridgecrest M7.1 2019", t="2019-07-06T03:19:53", lat=35.770, lon=-117.599, rad=1.2),
    dict(name="Ridgecrest M6.4 2019", t="2019-07-04T17:33:49", lat=35.705, lon=-117.504, rad=1.2),
    dict(name="La Habra M5.1 2014",   t="2014-03-29T04:09:42", lat=33.932, lon=-117.917, rad=0.8),
]

def run_detector(corr_input, sr):
    """Feed a vector-magnitude series (gal) through the faithful T2 detector.
    Returns dict with confirmation index t0 (or None) and the corrected series."""
    n = len(corr_input)
    baseline = corr_input[0]
    sta = 0.0
    lta = MIN_LTA_CLAMP
    corrected = np.empty(n)
    warm = int(WARMUP_S * sr)
    potential = False; pot_i = 0; in_event = False
    t0 = None; det_i = None
    for i in range(n):
        mag = corr_input[i]
        baseline = BASELINE_ALPHA*mag + (1-BASELINE_ALPHA)*baseline
        c = abs(mag - baseline); corrected[i] = c
        sta = STA_ALPHA*c + (1-STA_ALPHA)*sta
        if not in_event:
            lta = LTA_ALPHA*c + (1-LTA_ALPHA)*lta
        if i < warm:
            continue
        ratio = sta / max(lta, MIN_LTA_CLAMP)
        if not in_event and not potential:
            if ratio >= TRIG_RATIO and sta >= MIN_STA_GAL:
                potential = True; pot_i = i
        elif potential and not in_event:
            if ratio >= TRIG_RATIO and sta >= MIN_STA_GAL:
                if (i - pot_i)/sr >= CONFIRM_S:
                    in_event = True; t0 = i    # confirmation instant
            else:
                potential = False
        elif in_event:
            if ratio < DETRIG_RATIO or (i - t0)/sr > MAX_EVENT_S:
                det_i = i; break
    return dict(t0=t0, det_i=det_i, corrected=corrected, n=n)


def analyze_record(corrected, t0, det_i, sr):
    """Growth metrics from confirmation instant t0. Returns None if no confirmation."""
    if t0 is None:
        return None
    end = det_i if det_i is not None else len(corrected)
    end = max(end, t0 + 1)
    ev = corrected[t0:end]
    final = float(ev.max())
    prelim = float(corrected[t0])            # onset-instant sample = current PRELIM analog
    running = np.maximum.accumulate(ev)      # running peak from t0
    def peak_at(dt):
        k = min(int(round(dt*sr)), len(ev)-1)
        return float(running[k]), (k+1)/sr    # (running peak by t0+dt, actual dt covered)
    checks = {}
    for dt in CHECKPOINTS:
        if (len(ev)-1)/sr + 1e-9 < dt:       # event ended before this checkpoint
            checks[dt] = None; continue
        pk, _ = peak_at(dt)
        checks[dt] = dict(pga=pk, ratio_to_final=(pk/final if final>0 else None))
    # time (s from t0) to reach X% of final
    tpct = {}
    for p in (25,50,75,90):
        idx = np.argmax(running >= (p/100.0)*final)
        tpct[p] = float(idx/sr) if running[idx] >= (p/100.0)*final else None
    # time (s from t0) to first cross each threshold (instantaneous corrected, not running)
    tthr = {}
    for thr in THRESHOLDS:
        hits = np.nonzero(ev >= thr)[0]
        tthr[thr] = float(hits[0]/sr) if len(hits) else None
    return dict(final=final, prelim=prelim, dur_s=float(len(ev)/sr),
                checks=checks, t_to_pct=tpct, t_to_thr=tthr)


def vector_magnitude(st, sr):
    """3-comp accel stream (gal) -> sqrt(sum sq) magnitude series at sr Hz."""
    for tr in st:
        if abs(tr.stats.sampling_rate - sr) > 0.1:
            tr.resample(sr)
    t0 = max(tr.stats.starttime for tr in st)
    t1 = min(tr.stats.endtime for tr in st)
    if t1 <= t0:
        return None
    st.trim(t0, t1)
    m = min(len(tr.data) for tr in st)
    comps = [tr.data[:m].astype(float) for tr in st]
    return np.sqrt(sum(c*c for c in comps))


def fetch_event(client, ev, records, reasons):
    origin = UTCDateTime(ev["t"])
    t1, t2 = origin - (WARMUP_S + 40), origin + 120
    try:
        inv = client.get_stations(latitude=ev["lat"], longitude=ev["lon"], maxradius=ev["rad"],
                                  channel="HN?", starttime=t1, endtime=t2, level="response")
    except Exception as e:
        reasons.append(f"{ev['name']}: station query failed {type(e).__name__}"); return
    for net in inv:
        for sta in net:
            sid = f"{net.code}.{sta.code}"
            try:
                st = client.get_waveforms(net.code, sta.code, "*", "HN?", t1, t2, attach_response=False)
                if len(st) < 3:
                    reasons.append(f"{sid}: <3 comps"); continue
                st.merge(method=1, fill_value=0)
                st.remove_response(inventory=inv, output="ACC")
                for tr in st:
                    tr.data = tr.data * 100.0        # m/s^2 -> gal
                mag = vector_magnitude(st, SR)
                if mag is None or len(mag) < int((WARMUP_S+10)*SR):
                    reasons.append(f"{sid}: too short"); continue
                d = run_detector(mag, SR)
                res = analyze_record(d["corrected"], d["t0"], d["det_i"], SR)
                if res is None:
                    reasons.append(f"{sid}: no confirmation (T2 never triggered)"); continue
                dist_km = gps2dist_azimuth(ev["lat"], ev["lon"], sta.latitude, sta.longitude)[0]/1000.0
                res.update(event=ev["name"], station=sid, dist_km=round(dist_km,1))
                records.append(res)
            except Exception as e:
                reasons.append(f"{sid}: {type(e).__name__}"); continue


def pctl(vals, p):
    v = [x for x in vals if x is not None]
    return round(float(np.percentile(v, p)), 3) if v else None


def aggregate(records):
    n = len(records)
    out = {"n_records": n, "by_checkpoint": {}, "threshold_crossing": {}, "time_to_pct": {},
           "prelim_vs_later": {}}
    # ratio checkpoint/final distribution
    for dt in CHECKPOINTS:
        rs = [r["checks"][dt]["ratio_to_final"] for r in records if r["checks"].get(dt)]
        out["by_checkpoint"][dt] = {"n": len(rs), "median_ratio_to_final": pctl(rs,50),
                                    "p25": pctl(rs,25), "p75": pctl(rs,75),
                                    "min": pctl(rs,0), "max": pctl(rs,100)}
    # prelim (onset-instant) vs final ratio, for the two-window comparison
    pr = [r["prelim"]/r["final"] for r in records if r["final"]>0]
    out["prelim_vs_later"] = {"prelim_over_final_median": pctl(pr,50), "p25": pctl(pr,25),
                              "p75": pctl(pr,75),
                              "note": "compare against by_checkpoint ratios: how much a later "
                                      "window recovers of FINAL vs the current onset-instant PRELIM"}
    # threshold crossing: how many records reach thr by each checkpoint vs ever (final)
    for thr in THRESHOLDS:
        ever = sum(1 for r in records if r["final"] >= thr)
        by = {}
        for dt in CHECKPOINTS:
            c = sum(1 for r in records if r["checks"].get(dt) and r["checks"][dt]["pga"] >= thr)
            by[dt] = c
        late = ever - (by[CHECKPOINTS[0]] if CHECKPOINTS else 0)   # below at 0.3s but exceed later
        tt = [r["t_to_thr"][thr] for r in records if r["t_to_thr"][thr] is not None]
        out["threshold_crossing"][thr] = {"reach_ever(final>=thr)": ever, "reach_by_checkpoint": by,
            "below_at_0.3s_exceed_later": late,
            "time_to_cross_s": {"median": pctl(tt,50), "p25": pctl(tt,25), "p75": pctl(tt,75), "n": len(tt)}}
    for p in (25,50,75,90):
        ts = [r["t_to_pct"][p] for r in records if r["t_to_pct"][p] is not None]
        out["time_to_pct"][p] = {"median_s": pctl(ts,50), "p25": pctl(ts,25), "p75": pctl(ts,75), "n": len(ts)}
    return out


def main():
    outdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")
    os.makedirs(outdir, exist_ok=True)
    client = Client("IRIS", timeout=90)
    records, reasons = [], []
    for ev in EVENTS:
        print(f"[fetch] {ev['name']} ...", flush=True)
        fetch_event(client, ev, records, reasons)
        print(f"   records so far: {len(records)}  excluded: {len(reasons)}", flush=True)
    agg = aggregate(records)
    with open(os.path.join(outdir, "growth_records.json"), "w") as f:
        json.dump({"records": records, "excluded": reasons}, f, indent=1)
    with open(os.path.join(outdir, "growth_summary.json"), "w") as f:
        json.dump(agg, f, indent=1)
    print("\n=== SUMMARY ==="); print(json.dumps(agg, indent=1))
    print(f"\n{len(records)} records, {len(reasons)} excluded. out/ written.")

if __name__ == "__main__":
    main()



