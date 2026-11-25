# Official macOS diagnostic lab (not release packaging)

Dispatch existing Packaging workflow on the diagnostic branch with
`macos_diagnostics=true`. Runs on official `macos-15` (arm64) and
`macos-15-intel` (x64), not the owner's Mac. Does not create tags or releases.

1. Run Go tests with **native Core Text** and in-memory/software UI raster tests.
   Device ratios 1–3 in these tests are simulations, not attached Retina panels.
2. Build a clean ordinary executable before instrumentation and save build VCS
   identity/hash. External exact-PID NSRunningApplication termination is tested
   at initial scales 35/75/125%, using isolated runner-only app settings.
3. Inject CI-only readiness/draw/freeze hooks, save exact patch + generated Go,
   then build a distinctly named fixture. No diagnostic feature is compiled into
   ordinary candidate/release executables. Fixture is intentionally VCS-dirty.
4. Fixture uses real MyGo/Cocoa native windows and original close/quit handlers
   with SET never/open/hidden. Exercise framework main Close, NSWindow delegate
   windowShouldClose, NSApp terminate, actual NSMenu terminate-item action,
   synthetic in-process CmdQ menu key equivalent, and external native terminate.
   Menu-equivalent is NOT a physical keyboard test; delegate invocation is NOT a
   user clicking the close button. Frameless/accessory UI may have no close or
   Dock button. Tray's physical menu interaction is not tested.
5. Native-scale fixture logs live NSWindow content points, backing conversion,
   backingScaleFactor, actual UI draw rectangles and measured Core Text runs.
   Forced max/swell/juvenile states are fixtures, NOT real birth timing evidence.
6. `*-content-software.png` uses CapturePage's software/content render.
   Separately attempt `*-windowserver.png` with native screencapture of the
   fixture's own window number. Record success/permission errors; never disable
   screen-recording protections. Capture failure is not secretly a screenshot
   pass. A handshake binds each native capture to its logged scale/window ID.
7. Every quit requires exact spawned PID, exit=0, normal App.Run return plus all
   lifecycle milestones and PID gone. Timeout cleanup is failure, never a pass;
   only CI-spawned processes are cleaned up. Evidence uploads even on failure.

## Source/Apple semantics under investigation

MyGo v0.10.0 Cocoa `applicationShouldTerminate:` and its quit AppleEvent handler
run Go's QuitRequested, call backend Quit, then return NSTerminateCancel so
AppKit does not terminate Go before cleanup. Backend Quit uses NSApp stop:
**and posts an event** (or aborts a modal session first). Apple's stop: docs
explain that a timer/observer alone is insufficient without an NSEvent; this
wake event is already implemented, so do not patch in an unproven missing wake.
`applicationShouldTerminateAfterLastWindowClosed:` returns false. The Go layer,
not Cocoa's last-window policy, must explicitly request full application quit.
Native menu RoleQuit maps to `terminate:`; generic Go RoleQuit maps to App.Quit.

MyGo surface Size reports NSView bounds in points and backingScaleFactor as a
separate raster ratio. Core Text font Size/advances are in points/DIPs; glyph
rasterization scales once with the device ratio. Glyph keys include font identity
and 1/256 device scale. No evidence yet of a missing scale key or confirmed
Retina double-scaling. Native observations must decide, not Linux success.

Apple documentation consulted:
- https://developer.apple.com/documentation/appkit/nsapplication/terminate(_:)
- https://developer.apple.com/documentation/appkit/nsapplication/stop(_:)
- https://developer.apple.com/documentation/appkit/nswindow/performclose(_:)
- https://developer.apple.com/documentation/appkit/nswindow/backingscalefactor

Both owner's macOS defects remain OPEN until actual owner-Mac retest. These
runner checks do not close Gatekeeper/damaged-app, notarization, real CmdQ/tray/
Dock interaction, or real Retina/mixed-display behavior beyond observed backing
ratios. No certificates, keys or security policy changes are involved.
