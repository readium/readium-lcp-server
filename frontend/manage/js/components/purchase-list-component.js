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
var router_1 = require("@angular/router");
var common_1 = require("@angular/common");
var user_1 = require("./user");
var purchase_service_1 = require("./purchase.service");
var PurchasesComponent = (function () {
    function PurchasesComponent(purchaseService, route, location) {
        this.purchaseService = purchaseService;
        this.route = route;
        this.location = location;
    }
    PurchasesComponent.prototype.ngOnInit = function () {
        var _this = this;
        this.purchaseService.getPurchases(this.user)
            .then(function (purchases) { return _this.purchases = purchases; });
    };
    PurchasesComponent.prototype.goBack = function () {
        this.location.back();
    };
    PurchasesComponent.prototype.onSelect = function (p) {
        this.selectedPurchase = p;
    };
    PurchasesComponent.prototype.DownloadLicense = function (p) {
        // get License !
        if (p.licenseID === undefined) {
            console.log('Get license and download ' + p.label);
            // the license does not yet exist (some error occured ?)
            // we need to recontact the static server and ask to create a new license
            window.location.href = '/users/' + p.user.userID + '/purchases/' + p.purchaseID + '/license';
        }
        else {
            console.log('Re-download ' + p.label + '(' + p.licenseID + ')');
            // redirect to /licenses/ p.licenseID 
            window.location.href = '/licenses/' + p.licenseID;
        }
    };
    return PurchasesComponent;
}());
__decorate([
    core_1.Input(),
    __metadata("design:type", user_1.User)
], PurchasesComponent.prototype, "user", void 0);
PurchasesComponent = __decorate([
    core_1.Component({
        moduleId: module.id,
        selector: 'purchases',
        templateUrl: '/app/components/purchases.html',
        styleUrls: ['../../app/components/purchases.css'],
        providers: [purchase_service_1.PurchaseService]
    }),
    __metadata("design:paramtypes", [purchase_service_1.PurchaseService,
        router_1.ActivatedRoute,
        common_1.Location])
], PurchasesComponent);
exports.PurchasesComponent = PurchasesComponent;
//# sourceMappingURL=purchase-list-component.js.map