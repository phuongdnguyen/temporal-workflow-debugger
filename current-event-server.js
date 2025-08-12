const http = require('http');

const PORT = 30000;

const server = http.createServer((req, res) => {
  if (req.url === '/current-event') {
    console.log("Receive post request")
    const chunks = [];

    req.on('data', (chunk) => {
      chunks.push(chunk);
    });

    req.on('end', () => {
      const rawBody = Buffer.concat(chunks).toString();
      try {
        const parsed = JSON.parse(rawBody);
        console.log('[current-event] payload JSON:', parsed);
      } catch (_err) {
        console.log('[current-event] payload text:', rawBody);
      }
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ status: 'ok' }));
    });

    req.on('error', (err) => {
      console.error('Error reading request:', err);
      res.statusCode = 400;
      res.end('Bad Request');
    });
    return;
  }

  res.statusCode = 404;
  res.end('Not Found');
});

server.listen(PORT, () => {
  console.log(`Simple server listening on http://localhost:${PORT}`);
});


