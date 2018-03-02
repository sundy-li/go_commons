
var url = require('url');
var port, server, service,
    system = require('system');
var webpage = require('webpage');


function main() {
    if (system.args.length !== 2) {
        console.log('Usage: serverkeepalive.js <portnumber>');
        phantom.exit(1);
    } else {
        port = system.args[1];
        server = require('webserver').create();

        service = server.listen(port, { keepAlive: true }, function (request, response) {
            var url_parts = url.parse(request.url, true);
            var query = url_parts.query;
            if (query.hasOwnProperty('url')) {
                var query_url = decodeURIComponent(query['url'])
                var cookie = ""
                var ua = ""
                if (query.hasOwnProperty('cookie')) {
                    cookie = query['cookie']
                }

                if (query.hasOwnProperty('ua')) {
                    ua = query['ua']
                }

                console.log(query_url)
                console.log(cookie)
                console.log(ua)

                browser_query(query_url, cookie, ua)
            }

            var body = JSON.stringify(request, null, 4);
            response.statusCode = 200;
            response.headers = {
                'Cache': 'no-cache',
                'Content-Type': 'text/plain',
                'Connection': 'Keep-Alive',
                'Keep-Alive': 'timeout=5, max=100',
                'Content-Length': body.length
            };
            response.write(body);
            response.close();
        });

        if (service) {
            console.log('Web server running on port ' + port);
        } else {
            console.log('Error: Could not create web server listening on port ' + port);
            phantom.exit();
        }
    }
}

function browser_query(address, cookie, ua) {
    var page = webpage.create()
    if (ua.length > 0) {
        page.settings.userAgent = ua
    } else {
        page.settings.userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36"
    }

    
    page.onResourceRequested = function (req) {
        if (cookie.length > 0) {
            req.setHeader('Cookie', cookie)
        }
        console.log('requested: ' + JSON.stringify(req, undefined, 4));
    };

    page.onResourceReceived = function (res) {
        console.log('received: ' + JSON.stringify(res, undefined, 4));
    };

    t = Date.now();
    page.open(address, function (status) {
        if (status !== 'success') {
            console.log('FAIL to load the address');
        } else {
            setTimeout(function(){
                page.evaluate(function(){
                    if (Math.random() > 0.5) {
                        localStorage.clear();
                    }
                });
                page.close()
            }, 5000)
        }
    });
}

// run main
main()