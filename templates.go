package main

import (
	"html/template"
)

var indexTmpl = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html>
<h1>{{.Host}}</h1>
<ul>
{{range .Handlers}}<li><a href="https://{{.}}">{{.}}</a></li>{{end}}
</ul>
</html>
`))

var vanityTmpl = template.Must(template.New("vanity").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Import}} {{.VCS}} {{.Repo}}">
<meta name="go-source" content="{{.Import}} {{.Display}}">
<meta http-equiv="refresh" content="0; url={{.Repo}}">
</head>
<body>
Nothing to see here; <a href="{{.Repo}}">see at {{.Repo}}</a>.
</body>
</html>`))
