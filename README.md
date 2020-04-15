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

#### layout

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <!-- head things -->
    <style>
      {{ template "BaseCss" }}
    </style>
  </head>
  <body>
    <header>
      {{ template "LogoHtml" }}
      <!-- optional parts -->
    </header>

    <main>
      <! --
        main content
        see base.css
      -->
    </main>

    {{ template "FooterHtml" }}
  </body>
</html>
```

#### base css

```html
<style>
  {{ template "BaseCss" }}
</style>
```

assumes [layout](#layout)

#### header

```html
<header>
  {{ template "LogoHtml" }}
  <!-- optional parts
  <h2>A subtitle</h2>
  <p>A tagline</p>
  -->
</header>
```

#### footer

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

edit html / css files

```sh
go generate
```
