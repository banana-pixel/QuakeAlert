#!/usr/bin/env python3
# battery_capture.py — serial capture for the Admin Node disturbance battery
# (metric D, 07-threshold-comparison). READ-ONLY: it only reads the serial
# port and writes a local log. It never writes to the device or the server.
#
# Usage:
#   python3 battery_capture.py /dev/ttyUSB0 battery_2026-09-26.log
#
# Notes:
#  - Baud 115200 (firmware.ino:390 / platformio.ini:20).
#  - CH340 asserts DTR/RTS on open, so the node REBOOTS when this opens the
#    port (expected — see R-013). Wait 45 s (warmup) before the first tap.
#  - Line-buffered (buffering=1): a file object ignores `python -u`, which is
#    the 0-byte bug from R-017. Do not "fix" it back.
#  - Each line is prefixed with epoch-ms + monotonic-ms so disturbances can be
#    matched to your wall-clock announcements and to server rows later.
import sys, time
try:
    import serial  # pyserial
except ImportError:
    sys.exit("pyserial missing: pip install pyserial")

port = sys.argv[1] if len(sys.argv) > 1 else "/dev/ttyUSB0"
path = sys.argv[2] if len(sys.argv) > 2 else "battery_capture.log"

ser = serial.Serial()
ser.port = port
ser.baudrate = 115200
ser.timeout = 1
ser.dtr = False          # try not to reset; CH340 may reset anyway
ser.rts = False
ser.open()

t0 = time.time()
with open(path, "w", buffering=1) as f:
    f.write(f"# battery capture start epoch={t0:.3f} port={port}\n")
    print(f"capturing {port} -> {path} (start epoch {t0:.3f}); Ctrl-C to stop")
    try:
        while True:
            raw = ser.readline()
            if not raw:
                continue
            line = raw.decode("utf-8", "replace").rstrip("\r\n")
            stamp = f"{time.time():.3f} +{(time.time()-t0)*1000:.0f}ms"
            f.write(f"{stamp}\t{line}\n")
            print(f"{stamp}\t{line}")
    except KeyboardInterrupt:
        f.write(f"# battery capture end epoch={time.time():.3f}\n")
        print("\nstopped; footer written (clean finish)")
    finally:
        ser.close()
