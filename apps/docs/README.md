# fbt Docs Site

This directory contains the Astro/Starlight documentation site for `fbt`.

```sh
npm ci
SITE=https://nyuta01.github.io BASE=/fbt npm run build
npm run dev
```

The site is modeled after the Folio docs structure and is deployed through the
`release-docs` GitHub Actions workflow when GitHub Pages is enabled.
