# memegen

A simple Go command-line utility to generate a meme by adding outlined white text to an embedded template image (
`template.png`).

You'd typically use this together with something like pngcopy. 

## Usage

```bash
$ memegen -text 'Generate all the memes!!!'  | png2clip
```

### png2clip
This is on a mac. Probably a lot easier on Linux.
```bash
#!/bin/bash

# Create a temp file with .png extension
tmpfile=$(mktemp /tmp/clipboard_image.XXXXXX.png)

# Save stdin content to this temp file
cat > "$tmpfile"

# Use AppleScript to copy the image to clipboard
osascript -e "set the clipboard to (read (POSIX file \"$tmpfile\") as «class PNGf»)"

# Clean up temp file
rm "$tmpfile"
```


## Font Used

This tool embeds the **Bebas Neue** font (`font.ttf`).

- Copyright: Ryoichi Tsunekawa
- Source: <https://fonts.google.com/specimen/Bebas+Neue>
- License: SIL Open Font License (OFL)

## Requirements

- Go 1.16+

## Install
```bash
go install github.com/perbu/memegen@latest
```