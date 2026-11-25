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

Evidence so far:
- Close-policy unit matrix (tray on/off, SET never/open/hidden, native/session
  ending quit, canceled quit retry, save and active resize/input).
- Separate candidate 3cb184b Linux computerUse evidence: main-never PID344630,
  main-open PID345618, main-hidden PID349329, persistence PID353540 and settings
  snapshot PID358002 all have PASS_NATIVE_EXIT, exit=0, exact PID gone and the
  required lifecycle/run-return events. The executor read the frozen verdict
  and lifecycle files; no files in that candidate were altered. Parent reports
  SET 35%/45% persistence and restoration to 75% passed. This is real Linux
  native-loop evidence, not owner-platform acceptance or a test of this new
  display follow-up binary. Tray unavailable (libayatana missing); CtrlQ untested.
- Official-runner scripts require natural termination, exact child PID gone,
  exit=0 and `event=run-return`; not yet run for these local changes. Native
  macOS/Windows and keyboard CmdQ/Dock/tray interaction still need evidence.

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

Report: small-size display problem remains, with a fresh screenshot. A shared
copy was independently inspected, including nearest-neighbor enlarged crops.
The lower-left adult visibly reads `<o)))><` on one baseline, with its tail
near/over a green plant. The upper-right small run reads `((o>` on one baseline.
That exact run exists in the juvenile stage table (e.g. Size=.53), but the image
contains no fish ID/state or SET percentage. This pattern match does NOT prove
the fish is a juvenile or invalidate the owner's report. The still screenshot
cannot establish actual scale, DPI, installed version, dynamic-resize failure,
or what particular shape the owner expected. No OS is inferred from the image.

Investigation and separate local follow-up:
- The 35% minimum, font-size rounding/floor, weight cutoff and glow policy remain.
- Dynamic headless regression reuses a view through 125/75/55/35/55/35/75/125%
  with nominal device ratios 1, 1.25, 1.5, 2, 3 and body sizes .32/.5/.7/1/1.2;
  checks whole left/right sprite, single-line bounds, fresh-vs-reused layout.
  These pass on Linux; this is NOT native Windows/macOS raster acceptance.
- Additional software-rendered raster regression freezes simulation clocks and
  compares every glass-interior pixel between resized/re-DPI'd and fresh views,
  with an adult swell and a four-character juvenile run. This is a raster/cache
  check, not merely text-box measurement; native windows/other OSes still pending.
- Source audit: MyGo resolves generic monospace differently: Pango/fontconfig
  on Linux, monospaced system font (Menlo fallback) via Core Text on macOS,
  Cascadia Mono/Consolas/Courier New via DirectWrite on Windows. Fonts are not
  bundled, so exact widths and smoothing are platform-dependent.
- Layout identity includes text, family, DIP size, weight, NoWrap and width.
  Raster glyph keys include font identity and device scale (1/256), subpixel
  position, shade and stroke. Runtime repaints on size/device-scale changes.
  No missing-size or missing-DPI cache key was found; source review plus Linux
  tests do not rule out a native rasterizer or mixed-DPI defect.
- CONFIRMED SEPARATE BUG: maximum-size + birth-swell glyph exceeded the fixed
  adult physics box. At the right clamp, overrun was 11.2 DIP at 35%, 24 DIP at
  75% on Linux; the baseline regression failed on actual element bounds.
  Follow-up measures the whole run using the native text engine with precisely
  the draw family/size/weight, and only corrects overflowing draw coordinates
  inside the glass interior with glow clearance. It does not shrink growth,
  change physics/food/hover coordinates, clip characters, or raise minimum.
  Corner/scale/DPI/lifecycle/facing regressions now assert full visual containment.
  The historical physics box remains intentionally stable; visual/physics sizes
  are NOT claimed identical. Plant overlap inside the glass is NOT fixed by
  boundary containment, nor is this established as the screenshot's cause.
- Candidate 3cb184b and its GUI launcher/config/evidence remain untouched. This
  display follow-up needs its OWN GUI acceptance before any release decision.

Acceptance remaining:
- Comparable native before/after screenshots reproducing the owner's sequence.
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
