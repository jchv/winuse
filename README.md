# winuse
This is a small Go program, which can tell you what processes are actively using a set of files. It primarily exists to show off how one can bind to the Windows rstrtmgr API in Go.

```powershell
PS > go build .
PS > .\winuse.exe .\winuse.exe
processes:
- 5500 (AppName=winuse.exe)
reason: RmRebootReasonDetectedSelf
```
