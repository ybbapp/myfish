# Open user acceptance defects (GM-1)

These remain OPEN until the owner verifies both on their actual operating
system. The owner requested continued tracking (t43u). Their OS, display DPI,
and precise quit route have not yet been established. Never infer these from
an attached machine identifier. A Linux check, vanished window, successful
compilation, or a CI liveness/forced-cleanup test is not final acceptance.

## EXIT-RESIDUE — OPEN (t41u)

Report: quitting appears to leave the process running.

Confirmed baseline source behavior (preview.7):
- Main close with a tray on non-Linux explicitly hid the tank instead of exiting.
- App.OnQuitCanceled did not reset the application's own quitting flag. The
  framework resets its flag, so a later main close could skip a new quit request.
- These are real code defects/policies; neither is established as the precise
  cause of this user's instance because OS/route/native evidence is missing.

Local candidate: route main close through whole-application quit on all OSes;
prepare settings/input state before all native/window/menu quit paths; allow
framework-owned window destruction rather than recursively closing settings;
reset mirrored intent on cancellation. Add PID-tagged normal-loop-return logs.
Do not add force exit or change the framework fallback. Closing SET alone must
still hide only the SET window and allow continued use.

Evidence so far: close-policy unit matrix (tray on/off, SET never/open/hidden,
native/session-ending quit, canceled quit retry, save and active resize/input).
These do NOT establish native run-loop/PID termination. Official-runner scripts
now require natural termination, exact child PID disappearance, exit=0 and
`event=run-return`; they have not yet been run for this candidate. Keyboard
CmdQ/Dock and tray interaction still need native interaction evidence.

Acceptance remaining:
- Each main-close / menu-or-tray Quit / CtrlQ-or-CmdQ route across SET never
  opened, currently open, and previously opened then hidden.
- SET close alone leaves app alive, fish/FEED usable, SET can reopen.
- Normal before-quit, will-quit, quit-complete, run-return logs AND the exact
  launched PID gone with launcher exit=0; framework's 3-second force fallback
  without run-return is a FAIL even if PID disappears.
- Restart retains chosen scale/glass settings. Native macOS CmdQ and Dock Quit
  separately verified where available. Finally owner-platform reproduction
  and retest; do not close this defect on Linux/runner evidence alone.

## SMALL-DISPLAY — OPEN (t42u)

Report: small-size display problem remains, with a fresh screenshot. Executor
could not read the parent's private attachment; it has NOT independently
inspected that screenshot. Need an accessible shared copy before visual claims.
Do not label a partial fish string a font bug or dismiss it as an intended
juvenile without evidence. Keep minimum 35%, not a higher minimum workaround.

Investigation in local candidate:
- Production type rendering remains unchanged; no speculative display fix.
- Dynamic headless regression reuses a view through 125/75/55/35/55/35/75/125%
  with nominal device ratios 1, 1.25, 1.5, 2, 3 and body sizes .32/.5/.7/1/1.2;
  checks whole left/right sprite, single-line bounds, fresh-vs-reused layout.
  These pass on the Linux headless backend, NOT native Windows/macOS rasterizers
  or physical mixed-DPI displays.
- Diagnostic found a separate reproducible geometry mismatch: maximum-size +
  birth-swell visual glyph can extend beyond the fixed adult physics box. At
  the right clamp, overrun measured 11.2 DIP at 35%, 24 DIP at 75% on this backend.
  This is not established as the cause of the screenshot. No unreviewed change
  to fish growth, collision, or sprite bounds has been made to hide it.
- NoWrap, proportional stroke, small-font non-bold and 35% typography remain.
  Native font fallback, raster/cache behavior, dynamic scale/DPI transition and
  actual fish visual-scale/state at failure remain to be captured.

Acceptance remaining:
- Accessible screenshot inspection and comparable before/after screenshots.
- At 35%, left/right adult and juvenile characters coherent; distinguish fish
  states without presuming the report is expected lifecycle behavior.
- SET and edge resize repeatedly 35↔75↔125%, SET synchronization, normal 75%
  appearance and correct pointer coordinates, multiple device DPIs if available.
- Owner's actual system repeats the failing sequence and confirms resolution.

## Existing installation gate — still OPEN

macOS whole-bundle ad-hoc signing is NOT Developer ID/notarization; spctl refused
preview.7. Real browser-download Gatekeeper / “damaged app” is not closed. No
certificates bought, private keys read, or security policies disabled. Windows
unsigned Setup does not imply SmartScreen acceptance. ARM64 install and real-user
interactive upgrade/uninstall remain pending. New natural-quit runner checks are
an additional gate, not retroactive acceptance of prior forced-cleanup runs.
