# App shell layout

```
index.html
└── #app-root.app
    ├── components/sidebar.html
    └── .main
        ├── components/topbar.html
        ├── main.content#page-root
        │   └── pages/*.html (one section per route)
        └── components/flow-banner.html
```

Navigation uses hash routes (`#strategic-dashboard`, `#host/prod-db-02`, etc.).
`scripts/app/prototype-app.js` owns page interactions; `scripts/router/` holds route metadata.
