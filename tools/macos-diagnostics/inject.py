"""Instrument CI checkout only; never applied to release/candidate binaries.

The ordinary clean binary is built first. Save git diff + generated Go file as
acceptance evidence. No runtime diagnostic switch is added to production code.
"""
from pathlib import Path
root = Path(__file__).resolve().parents[2]
p = root / 'main.go'
s = p.read_text()
a = 'mygo.App.WhenReady(a.ready)'
b = '''mygo.App.WhenReady(func() {
        mygo.App.SetPath(mygo.PathUserData, filepath.Join(os.Getenv("MYFISH_DIAG_DIR"), "config"))
        a.ready()
        startMacDiagnostics(a)
    })'''
assert s.count(a) == 1
s = s.replace(a, b).replace('"log"', '"log"\n "os"\n "path/filepath"', 1)
a = 'now := c.Now()'
assert s.count(a) == 1
s = s.replace(a, a + '\n if macDiagFreeze { a.last = now }', 1)
p.write_text(s)
p = root / 'look.go'
s = p.read_text()
a = '.Top(float32(dipY(y, scale)))'
assert s.count(a) == 1
s = s.replace(a, a + '.DrawOver(func(_ *ui.Painter, r ui.Rect) { recordMacFishBounds(key, s, size, scale, r) })', 1)
p.write_text(s)
(root/'zz_mac_diagnostic.go').write_text((Path(__file__).parent/'fixture.go.txt').read_text())
