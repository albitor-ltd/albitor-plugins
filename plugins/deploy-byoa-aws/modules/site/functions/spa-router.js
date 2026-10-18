// spa-router: CloudFront Function (cloudfront-js-2.0), viewer-request, default behaviour only.
//
// A single-page app needs a deep link such as /dashboard/42 to serve the index document, because no
// such object exists in S3. A distribution custom_error_response would do it, but it is
// distribution-wide: it also turns /api/<unknown> and a missing /app.js into `200 text/html`. This
// function does the rewrite where it can be scoped.
//
// A path under /api/ is never rewritten (the API has its own behaviour; with no API the request
// reaches S3 and fails honestly). A last segment containing a dot is a file, and a missing file must
// fail, not return the shell. Everything else is an application route and serves the index.
//
// The cost: a route whose last segment has a dot (/users/jane.doe) is taken for a file and fails.
// Listing known asset extensions instead was rejected, because a real file with an unlisted
// extension would then be rewritten to the shell. The plugin README tells apps to keep dots out of
// the last segment of a route.
function handler(event) {
  var request = event.request;
  var uri = request.uri;

  if (uri === '/api' || uri.indexOf('/api/') === 0) {
    return request;
  }

  var lastSegment = uri.substring(uri.lastIndexOf('/') + 1);
  if (lastSegment.indexOf('.') !== -1) {
    return request;
  }

  request.uri = '/${index_path}';
  return request;
}
