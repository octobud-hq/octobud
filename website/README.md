# Octobud Landing Page

This is the static landing page for [octobud.io](https://octobud.io).

## Files

- `index.html` - Main landing page
- `logo.svg` - Octobud logo
- `favicon.png` - Browser favicon (copy from `frontend/static/favicon.png`)

## Deployment

This is a static site that can be deployed to any static hosting service:

### GitHub Pages

1. Push to a `gh-pages` branch, or
2. Use a GitHub Action to deploy to GitHub Pages

### Vercel / Netlify

1. Connect your repository
2. Set the root directory to `website/`
3. Deploy

### Manual

Simply upload the files to any web server or CDN.

## Development

To preview locally, you can use any static file server:

```bash
# Using Python
cd website
python -m http.server 8000

# Using Node.js (npx)
npx serve website

# Using PHP
cd website
php -S localhost:8000
```

Then open http://localhost:8000 in your browser.

