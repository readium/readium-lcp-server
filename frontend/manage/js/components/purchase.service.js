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
var purchase_1 = require("./purchase");
var PurchaseService = (function () {
    function PurchaseService(http) {
        this.http = http;
        this.usersUrl = Config.frontend.url + '/users';
        // /users/{user_id}/purchases
        this.headers = new http_1.Headers({ 'Content-Type': 'application/json' });
    }
    PurchaseService.prototype.getPurchases = function (user) {
        return this.http.get(this.usersUrl + '/' + user.userID + '/purchases')
            .toPromise()
            .then(function (response) {
            var purchases = [];
            for (var _i = 0, _a = response.json(); _i < _a.length; _i++) {
                var ResponseItem = _a[_i];
                var p = new purchase_1.Purchase;
                p.label = ResponseItem.label;
                p.licenseID = ResponseItem.licenseID;
                p.purchaseID = ResponseItem.purchaseID;
                p.resource = ResponseItem.resource;
                p.transactionDate = ResponseItem.transactionDate;
                p.user = ResponseItem.user;
                p.partialLicense = ResponseItem.partialLicense;
                purchases[purchases.length] = p;
            }
            return purchases;
        })
            .catch(this.handleError);
    };
    PurchaseService.prototype.create = function (purchase) {
        return this.http
            .put(this.usersUrl + '/' + purchase.user.userID + '/purchases', JSON.stringify(purchase), { headers: this.headers })
            .toPromise()
            .then(function (response) {
            if ((response.status === 200) || (response.status === 201)) {
                return purchase; // ok
            }
            else {
                throw 'Error in create(purchase); ' + response.status + response.text;
            }
        })
            .catch(this.handleError);
    };
    PurchaseService.prototype.handleError = function (error) {
        console.error('An error occurred (purchase-service)', error);
        return Promise.reject(error.message || error);
    };
    return PurchaseService;
}());
PurchaseService = __decorate([
    core_1.Injectable(),
    __metadata("design:paramtypes", [http_1.Http])
], PurchaseService);
exports.PurchaseService = PurchaseService;
//# sourceMappingURL=purchase.service.js.map