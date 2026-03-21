# React + TypeScript + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Oxc](https://oxc.rs)
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/)

# Eventra Frontend

Eventra frontend is a premium auth console UI for the Go backend.

## Features

- Register
- Login
- Refresh token rotation
- Logout with refresh token revoke
- Protected me endpoint integration
- Backend health indicator

## Setup

1. Copy .env.example to .env
2. Set API base URL if needed

Default value:

VITE_API_BASE_URL=http://localhost:8080

## Run

Install dependencies:

npm install

Start dev server:

npm run dev

Build for production:

npm run build
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```
