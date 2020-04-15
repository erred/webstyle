# webstyle

common css styles embedded in go templates

[![License](https://img.shields.io/github/license/seankhliao/webstyle.svg?style=flat-square)](LICENSE)
![Version](https://img.shields.io/github/v/tag/seankhliao/webstyle?sort=semver&style=flat-square)
[![pkg.go.dev](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/go.seankhliao.com/webstyle)

## usage

```go
import "go.seankhliao.com/webstyle"
```

### components

##### base css

```html
<style>
  {{ template "BaseCss" }}
</style>
```

assumes [layout](#layout)

##### fonts

```html
<style>
  {{ template "FontsCss" }}
</style>
```

##### layout

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    {{ template "HeadGohtml" . }}
  </head>
  <body>
    <header>
      {{ template "LogoHtml" }}
      <!-- optiona parts -->
    </header>

    <main>
      {{ .Main }}
    </main>

    {{ template "FooterHtml" }}
  </body>
</html>
```

##### head

##### header

```html
<header>
  {{ template "LogoHtml" }}
  <!-- optional
  <h2>A subtitle</h2>
  <p>A tagline</p>
  -->
</header>
```

##### footer

```html
{{ template "FooterHtml" }}
```

#### loader

```html
<style>
  {{ template "LoaderCss" }}
</style>

{{ template "LoaderHtml" }}
```

## develop

```sh
go generate
```
