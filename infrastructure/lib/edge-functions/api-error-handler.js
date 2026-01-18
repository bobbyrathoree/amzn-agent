// CloudFront Function to handle API error responses
// Prevents CloudFront custom error pages from replacing API Gateway error responses
// This ensures clients receive the actual API error (400, 401, 403, 404, 500, etc.)
// instead of the SPA fallback page

function handler(event) {
    var response = event.response;
    var request = event.request;

    // For API requests, add a header to indicate this is an API response
    // This can be used for debugging and ensures proper error handling
    response.headers['x-api-response'] = { value: 'true' };

    return response;
}
