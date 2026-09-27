#!/usr/bin/env python3
# research-only smoke test — offline waveform replay feasibility (task 7c8d / doc 16 §6)
# NOT production code. Read-only: fetches public FDSN data, writes nothing back.
# Proves: FDSN fetch + StationXML response removal + PGA-in-gal works from this host.
import sys
from obspy import UTCDateTime
from obspy.clients.fdsn import Client

# 2019-07-06 Ridgecrest M7.1 mainshock (well-documented, open FDSN data).
ORIGIN = UTCDateTime("2019-07-06T03:19:53")
EPI_LAT, EPI_LON = 35.770, -117.599            # Ridgecrest M7.1 epicenter
CHA = "HN?"                                     # strong-motion accelerometer, ~100 sps

def main():
    c = Client("IRIS", timeout=60)
    t1, t2 = ORIGIN - 10, ORIGIN + 120
    inv = c.get_stations(latitude=EPI_LAT, longitude=EPI_LON, maxradius=0.6,
                         channel=CHA, starttime=t1, endtime=t2, level="response")
    cand = [(n.code, s.code) for n in inv for s in n]
    print(f"{len(cand)} candidate stations w/ {CHA}: {cand[:12]}")
    for net, sta in cand:
        try:
            st = c.get_waveforms(net, sta, "*", CHA, t1, t2)
        except Exception:
            continue
        if not st:
            continue
        st.remove_response(inventory=inv, output="ACC")   # -> m/s^2
        print(f"GOT {net}.{sta}:")
        for tr in st:
            pga = float(max(abs(tr.data.min()), abs(tr.data.max())))
            print(f"  {tr.id}: sr={tr.stats.sampling_rate:.1f}Hz npts={tr.stats.npts} "
                  f"PGA={pga:.4f} m/s^2 = {pga*100:.2f} gal")
        print("SMOKE OK: FDSN fetch + response removal + gal conversion works.")
        return
    print("SMOKE FAILED: no candidate station returned data.")
    sys.exit(1)

if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print("SMOKE FAILED:", type(e).__name__, e); sys.exit(1)
