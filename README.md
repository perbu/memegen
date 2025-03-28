# meme-gen

A simple Go command-line utility to generate a meme by adding outlined white text to an embedded template image (
`template.png`).

You'd typically use this together with something like pngcopy. 

## Usage

```bash
$ meme-gen -text 'Generate all the memes!!!'  | pngcopy
```

## Font Used

This tool embeds the **Bebas Neue** font (`font.ttf`).

- Copyright: Ryoichi Tsunekawa
- Source: <https://fonts.google.com/specimen/Bebas+Neue>
- License: SIL Open Font License (OFL)

## Requirements

- Go 1.16+

## Build

```bash
go build -o meme-gen .
```