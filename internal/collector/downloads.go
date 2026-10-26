package collector

import (
	"path"
	"strings"
)

type Downloads struct{}

func (c *Downloads) Name() string { return "downloads" }

var execExt = map[string]bool{
	".exe": true, ".msi": true, ".dmg": true, ".pkg": true,
	".bat": true, ".cmd": true, ".sh": true, ".ps1": true,
	".js": true, ".vbs": true, ".jar": true, ".apk": true,
	".deb": true, ".rpm": true, ".com": true, ".scr": true,
}

func (c *Downloads) Collect(ctx *Context) (interface{}, error) {
	var items []map[string]interface{}
	if ctx.Tracker != nil {
		items = ctx.Tracker.Downloads()
	}

	names := []string{}
	types := []string{}
	execDetected := false

	for _, d := range items {
		fn, _ := d["filename"].(string)
		if fn == "" {
			continue
		}
		names = append(names, fn)
		ext := strings.ToLower(path.Ext(fn))
		if ext != "" {
			types = append(types, ext)
			if execExt[ext] {
				execDetected = true
			}
		}
	}

	return map[string]interface{}{
		"downloadAttemptCount":       len(items),
		"downloadedFileNames":        names,
		"downloadedFileTypes":        types,
		"executableDownloadDetection": execDetected,
	}, nil
}
