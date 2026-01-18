// CloudFront Function to rewrite API paths
// Strips the environment prefix (e.g., /dev/, /prod/) from the request URI
// before forwarding to API Gateway origin

function handler(event) {
    var request = event.request;
    var uri = request.uri;

    // Match pattern: /{env}/path -> /path
    // This strips the first path segment (environment prefix)
    var match = uri.match(/^\/[^\/]+(\/.*)$/);

    if (match) {
        request.uri = match[1];
    }

    return request;
}
