// Copyright 2026 The Caslingo Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package routers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/beego/beego/v2/server/web/context"
	"github.com/caslingo/caslingo/conf"
)

// Set explicitly: on Windows, Go reads MIME types from the registry, which sometimes maps .js to text/plain and breaks ES modules.
var contentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".jpg":   "image/jpeg",
	".png":   "image/png",
	".svg":   "image/svg+xml",
	".woff2": "font/woff2",
	".ico":   "image/x-icon",
}

// StaticFilter serves the built frontend (webDir), a React single-page app: paths that are not files get index.html.
func StaticFilter(ctx *context.Context) {
	urlPath := ctx.Request.URL.Path
	if strings.HasPrefix(urlPath, "/api/") {
		return
	}
	if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead {
		return
	}

	relPath := strings.TrimPrefix(urlPath, "/")
	if !filepath.IsLocal(relPath) && relPath != "" {
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}

	webDir := conf.GetConfigString("webDir")
	path := filepath.Join(webDir, relPath)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		if filepath.Ext(relPath) != "" && relPath != "index.html" {
			http.NotFound(ctx.ResponseWriter, ctx.Request)
			return
		}
		path = filepath.Join(webDir, "index.html")
	}
	serveFile(ctx, path, strings.HasPrefix(relPath, "assets/"))
}

func serveFile(ctx *context.Context, path string, immutable bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		if strings.HasSuffix(path, "index.html") {
			http.Error(ctx.ResponseWriter, "frontend not built: run `yarn install && yarn build` in web/", http.StatusNotFound)
			return
		}
		http.NotFound(ctx.ResponseWriter, ctx.Request)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		http.Error(ctx.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(path))
	header := ctx.ResponseWriter.Header()
	if contentType, ok := contentTypes[ext]; ok {
		header.Set("Content-Type", contentType)
	}
	if immutable {
		// Vite puts a content hash in every file name under assets/
		header.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		header.Set("Cache-Control", "no-cache")
	}

	http.ServeContent(ctx.ResponseWriter, ctx.Request, info.Name(), info.ModTime(), file)
}
