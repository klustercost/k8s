# klustercost — Presentation Website

The marketing and presentation site for **klustercost**, a single-page static website built with HTML, Tailwind CSS, and vanilla JavaScript.

## Quick Start

No build step required. Open the site directly or serve it with any HTTP server:

```bash
# Python
cd website
python -m http.server 8888

# Node
npx serve website

# Then open http://localhost:8888
```

## Project Structure

```
website/
├── index.html          Single-page site (all 9 sections)
├── css/
│   └── styles.css      Custom animations, glassmorphism, gradients
├── js/
│   └── main.js         Scroll animations, typing effect, interactions
├── assets/             Static assets (SVG icons are inline)
├── Dockerfile          Production container (nginx:alpine)
├── nginx.conf          Gzip, caching, security headers
└── README.md
```

## Tech Stack

| Layer       | Technology                                     |
| ----------- | ---------------------------------------------- |
| Markup      | HTML5, semantic sections                       |
| Styling     | Tailwind CSS v3 (CDN), custom CSS              |
| Typography  | Inter (body), JetBrains Mono (code)            |
| JavaScript  | Vanilla JS — no frameworks, no bundler         |
| Deployment  | nginx:alpine Docker image                      |

## Page Sections

1. **Navigation** — sticky, transparent-to-blur on scroll, mobile hamburger menu
2. **Hero** — full-viewport with animated dot-grid background, terminal preview
3. **Problem Statement** — three pain-point cards (unpredictable bills, no per-pod visibility, manual pricing)
4. **Features** — six glassmorphism cards covering monitoring, pricing, AI, WhatsApp, Power BI, Helm
5. **How It Works** — four-step architecture flow with animated connection line
6. **AI-Powered Queries** — live typing animation cycling through real-world questions with JSON responses
7. **Quick Start** — three-step install guide with copy-to-clipboard code blocks
8. **Open Source CTA** — GitHub star and issue links
9. **Footer** — navigation links, branding, license

## Design

- **Dark theme** — slate-950 base with cyan-to-blue gradient accents
- **Glassmorphism** — semi-transparent cards with backdrop blur
- **Scroll animations** — elements fade and slide in via IntersectionObserver
- **Typing effect** — AI section cycles through example queries automatically
- **Responsive** — mobile-first with breakpoints at `sm`, `md`, and `lg`
- **Accessibility** — respects `prefers-reduced-motion`, semantic HTML, proper contrast

## Docker Deployment

Build and run the production container:

```bash
cd website

docker build -t ghcr.io/klustercost/k8s/klustercost-website:latest .
docker run -p 80:80 ghcr.io/klustercost/k8s/klustercost-website:latest
```

The nginx configuration includes gzip compression, 30-day cache headers for static assets, and security headers (`X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`).

## Kubernetes Deployment

To serve the website from your cluster, create a Deployment and Service:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: klustercost-website
spec:
  replicas: 1
  selector:
    matchLabels:
      app: klustercost-website
  template:
    metadata:
      labels:
        app: klustercost-website
    spec:
      containers:
        - name: website
          image: ghcr.io/klustercost/k8s/klustercost-website:latest
          ports:
            - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: klustercost-website
spec:
  type: LoadBalancer
  selector:
    app: klustercost-website
  ports:
    - port: 80
      targetPort: 80
```

## Customization

| What                    | Where                          |
| ----------------------- | ------------------------------ |
| Content and copy        | `index.html`                   |
| Colors and animations   | `css/styles.css`               |
| Tailwind theme          | `<script>` block in `<head>`   |
| Interactions            | `js/main.js`                   |
| GitHub / repo URLs      | Search `github.com/klustercost` in `index.html` |

All styling uses Tailwind utility classes loaded from the CDN — no build pipeline to configure. Custom CSS is limited to effects that Tailwind cannot express inline (keyframe animations, backdrop-filter, mask-image).
