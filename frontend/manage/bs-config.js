var vm = require("vm");
var fs = require("fs");

var configJS = fs.readFileSync("./config.js");

global.window = {};
vm.runInThisContext(configJS);
// eval() has direct access to the local context, no need for global.window = 
//eval(configJS);

var url = global.window.Config.frontend.url;
console.log(url);
var ip = url.replace("http://", "").replace("https://", "");
var i = ip.indexOf(":");
if (i > 0) {
  ip = ip.substr(0, i);
}
console.log(ip);

module.exports = {
  "baseDir": "./",
  "port": 3000,
  "files": ["./**/*.{html,htm,css,js}"],
  //"logLevel": "debug",
  "logPrefix": "Readium BS",
  "logConnections": true,
  "logFileChanges": true,
  "open": "external",
  "host": ip
};