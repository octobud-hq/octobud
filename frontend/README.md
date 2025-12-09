# Octobud Frontend

SvelteKit frontend for Octobud.

## Quick Start

### From Root Directory
```bash
# First-time setup (from project root)
make frontend-install

# Run frontend dev server
make frontend-dev
```

### From Frontend Directory
```bash
# Install dependencies
npm install

# Run dev server
npm run dev
```

The dev server will start at http://localhost:5173

## Development

### API Proxy Configuration

The frontend proxies API requests to the backend server. Configuration is in `vite.config.ts`:

```typescript
proxy: {
  '/api': {
    target: 'http://localhost:8080',  // Backend server
    changeOrigin: true,
  }
}
```

You can override this with the `VITE_API_PROXY_TARGET` environment variable:

```bash
VITE_API_PROXY_TARGET=http://localhost:9000 npm run dev
```

### Hot Module Replacement

Vite provides instant hot module replacement. Changes to `.svelte` files will automatically update in your browser without losing state.

## Building for Production

```bash
npm run build
```

The production build will be output to the `build/` directory.

### Preview Production Build

```bash
npm run preview
```

## Project Structure

```
src/
├── lib/
│   ├── api/              # API client functions
│   ├── components/       # Svelte components
│   ├── constants/        # Constants and configuration
│   ├── keyboard/         # Keyboard shortcut handlers
│   ├── state/            # State management
│   ├── stores/           # Svelte stores
│   └── utils/            # Utility functions
└── routes/
    ├── +layout.svelte    # Root layout
    ├── +page.svelte      # Home page
    └── views/            # View-specific routes
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run check` - Run type checking
- `npm run lint` - Run ESLint
- `npm test` - Run tests
- `npm run test:watch` - Run tests in watch mode

## Tech Stack

- **SvelteKit** - Web framework
- **Vite** - Build tool and dev server
- **TypeScript** - Type safety
- **Tailwind CSS** - Styling
- **Skeleton UI** - Component library

## Troubleshooting

### Port Already in Use
If port 5173 is in use:
```bash
# Use a different port
npm run dev -- --port 5174
```

### Backend Connection Issues
Ensure the backend server is running at http://localhost:8080:
```bash
# From project root
make backend-dev
```

### Module Not Found Errors
```bash
# Clear and reinstall dependencies
rm -rf node_modules package-lock.json
npm install
```

### Type Errors
Run the type checker to see detailed errors:
```bash
npm run check
```

