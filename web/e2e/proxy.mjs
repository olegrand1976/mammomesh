// Remplaçant minimal de Caddy pour les tests e2e : une seule origine,
// /api et /mcp vers l'API Go, le reste vers le serveur Nuxt. Le flux SSE passe sans tampon.
import http from 'node:http'

const port = Number(process.env.E2E_PORT ?? 14000)
const api = new URL(process.env.E2E_API ?? 'http://127.0.0.1:18080')
const web = new URL(process.env.E2E_WEB ?? 'http://127.0.0.1:13000')

http.createServer((req, res) => {
  const target = /^\/(api|mcp)(\/|\?|$)/.test(req.url ?? '') ? api : web
  const up = http.request({ hostname: target.hostname, port: target.port, path: req.url, method: req.method, headers: req.headers }, (r) => {
    res.writeHead(r.statusCode ?? 502, r.headers)
    r.pipe(res)
  })
  up.on('error', () => { if (!res.headersSent) res.writeHead(502); res.end() })
  req.pipe(up)
}).listen(port, '127.0.0.1', () => console.log(`proxy e2e prêt sur http://127.0.0.1:${port}`))
