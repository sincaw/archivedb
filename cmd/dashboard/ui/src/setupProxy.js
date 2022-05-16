const proxy = require('http-proxy-middleware')

module.exports = (app) => {
  app.use(proxy.createProxyMiddleware(['/api'], {
    target: process.env.PROXY || 'http://127.0.0.1:8000', changeOrigin: true, secure: false
  }))
}
