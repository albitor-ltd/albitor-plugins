// Starting point (web-svelte): the web side of the local run contract.
//
// Serves the built SPA from ./build with the single-page fallback, forwards /api/* to the API
// (the api-runtime plugin starts it on :8080 in local mode), and answers /healthz so the runner
// can wait for it. `npm run local` builds first, then runs this.
//
//   PORT                 listen port (default 3000)
//   APP_API_ORIGIN   where /api goes (default http://127.0.0.1:8080)
import { createReadStream, statSync } from 'node:fs';
import http from 'node:http';
import https from 'node:https';
import { extname, join, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(fileURLToPath(new URL('../build', import.meta.url)));
const port = Number(process.env.PORT ?? 3000);
const api = new URL(process.env.APP_API_ORIGIN ?? 'http://127.0.0.1:8080');

const types = {
	'.html': 'text/html; charset=utf-8',
	'.js': 'text/javascript; charset=utf-8',
	'.mjs': 'text/javascript; charset=utf-8',
	'.css': 'text/css; charset=utf-8',
	'.json': 'application/json; charset=utf-8',
	'.svg': 'image/svg+xml',
	'.png': 'image/png',
	'.jpg': 'image/jpeg',
	'.ico': 'image/x-icon',
	'.woff2': 'font/woff2',
	'.txt': 'text/plain; charset=utf-8',
	'.webmanifest': 'application/manifest+json'
};

function fileAt(pathname) {
	let decoded;
	try {
		decoded = decodeURIComponent(pathname);
	} catch {
		return null;
	}
	const target = resolve(join(root, decoded));
	if (target !== root && !target.startsWith(root + sep)) return null;
	try {
		return statSync(target).isFile() ? target : null;
	} catch {
		return null;
	}
}

function proxy(req, res) {
	const transport = api.protocol === 'https:' ? https : http;
	const upstream = transport.request(
		{
			protocol: api.protocol,
			hostname: api.hostname,
			port: api.port,
			path: req.url,
			method: req.method,
			headers: { ...req.headers, host: api.host }
		},
		(up) => {
			res.writeHead(up.statusCode ?? 502, up.headers);
			up.pipe(res);
		}
	);
	upstream.on('error', () => {
		if (!res.headersSent) res.writeHead(502, { 'Content-Type': 'text/plain' });
		res.end('the API is not reachable');
	});
	req.pipe(upstream);
}

const hostOk = (host = '') => /^(localhost|127\.0\.0\.1)(:\d+)?$/i.test(host);

const server = http.createServer((req, res) => {
	// Refuse DNS-rebinding hosts and absolute-form request targets (`GET http://evil/ HTTP/1.1`).
	const target = req.url ?? '/';
	if (!hostOk(req.headers.host) || !target.startsWith('/') || target.startsWith('//')) {
		res.writeHead(400, { 'Content-Type': 'text/plain' });
		res.end('bad request');
		return;
	}
	const { pathname } = new URL(req.url ?? '/', 'http://localhost');
	if (pathname === '/healthz') {
		res.writeHead(200, { 'Content-Type': 'text/plain' });
		res.end('ok');
		return;
	}
	if (pathname === '/api' || pathname.startsWith('/api/')) {
		proxy(req, res);
		return;
	}
	if (req.method !== 'GET' && req.method !== 'HEAD') {
		res.writeHead(405, { Allow: 'GET, HEAD' });
		res.end();
		return;
	}
	const file = fileAt(pathname);
	const served = file ?? fileAt('/index.html');
	if (!served) {
		res.writeHead(404, { 'Content-Type': 'text/plain' });
		res.end('build the app first: npm run build');
		return;
	}
	res.writeHead(200, {
		'Content-Type': types[extname(served)] ?? 'application/octet-stream',
		'Cache-Control': pathname.includes('/_app/immutable/')
			? 'public, max-age=31536000, immutable'
			: 'no-cache'
	});
	if (req.method === 'HEAD') res.end();
	else createReadStream(served).pipe(res);
});

server.listen(port, '127.0.0.1', () => {
	console.log(`web listening on http://127.0.0.1:${port}, /api -> ${api.origin}`);
});
