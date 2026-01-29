const http = require("http");

const PORT = 3000;

const server = http.createServer((req, res) => {
  if (req.url === "/" || req.url === "/index.html") {
    console.log("Hello World");
    res.writeHead(200, { "Content-Type": "text/plain" });
    res.end("Hello World\n");
  } else {
    res.writeHead(404, { "Content-Type": "text/plain" });
    res.end("Not Found\n");
  }
});

server.listen(PORT, () => {
  console.log(`Server running at http://localhost:${PORT}/`);
});
