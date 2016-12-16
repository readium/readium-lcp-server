"use strict";
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __metadata = (this && this.__metadata) || function (k, v) {
    if (typeof Reflect === "object" && typeof Reflect.metadata === "function") return Reflect.metadata(k, v);
};
var core_1 = require("@angular/core");
var http_1 = require("@angular/http");
require("rxjs/add/operator/toPromise");
var CryptoJS = require("angular-crypto-js");
var UserService = (function () {
    function UserService(http) {
        this.http = http;
        this.usersUrl = 'http://localhost/users'; // THIS SHOULD BE EQUAL TO THE URL of the static webserver (or just /)
        this.headers = new http_1.Headers({ 'Content-Type': 'application/json' });
    }
    UserService.prototype.getUsers = function () {
        return this.http.get(this.usersUrl)
            .toPromise()
            .then(function (response) {
            var users = [];
            for (var _i = 0, _a = response.json(); _i < _a.length; _i++) {
                var jsonUser = _a[_i];
                users[users.length] = { userID: jsonUser.userID, alias: jsonUser.alias, email: jsonUser.email, password: jsonUser.password };
            }
            return users;
        })
            .catch(this.handleError);
    };
    UserService.prototype.create = function (newAlias, newEmail, newPassword) {
        var hash = CryptoJS.createHash('sha256');
        hash.update(newPassword);
        var hashedPassword = hash.digest('hex');
        var user = { userID: null, alias: newAlias, email: newEmail, password: hashedPassword };
        return this.http
            .put(this.usersUrl, JSON.stringify(user), { headers: this.headers })
            .toPromise()
            .then(function (response) {
            if (response.status === 201) {
                return user;
            }
            else {
                throw 'Error creating user ' + response.text;
            }
        })
            .catch(this.handleError);
    };
    UserService.prototype.delete = function (id) {
        var url = this.usersUrl + "/" + id;
        return this.http.delete(url, { headers: this.headers })
            .toPromise()
            .then(function () { return null; })
            .catch(this.handleError);
    };
    UserService.prototype.handleError = function (error) {
        console.error('An error occurred', error);
        return Promise.reject(error.message || error);
    };
    UserService.prototype.getUser = function (id) {
        return this.getUsers()
            .then(function (users) { return users.find(function (user) { return user.userID === id; }); });
    };
    UserService.prototype.update = function (user) {
        var url = this.usersUrl + "/" + user.userID;
        return this.http
            .put(url, JSON.stringify(user), { headers: this.headers })
            .toPromise()
            .then(function () { return user; })
            .catch(this.handleError);
    };
    return UserService;
}());
UserService = __decorate([
    core_1.Injectable(),
    __metadata("design:paramtypes", [http_1.Http])
], UserService);
exports.UserService = UserService;
//# sourceMappingURL=user.service.js.map