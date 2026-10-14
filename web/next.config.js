/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: process.env.API_BASE_URL
          ? `${process.env.API_BASE_URL}/api/v1/:path*`
          : 'http://localhost:8080/api/v1/:path*',
      },
      {
        source: '/healthz',
        destination: process.env.API_BASE_URL
          ? `${process.env.API_BASE_URL}/healthz`
          : 'http://localhost:8080/healthz',
      },
    ];
  },
  output: 'standalone',
};

module.exports = nextConfig;
