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
var user_1 = require("./user");
var purchase_1 = require("./purchase");
var lic = require("./partialLicense");
var resource_service_1 = require("./resource.service");
var purchase_service_1 = require("./purchase.service");
var ResourcesComponent = (function () {
    function ResourcesComponent(resourceService, purchaseService, router) {
        this.resourceService = resourceService;
        this.purchaseService = purchaseService;
        this.router = router;
    }
    ResourcesComponent.prototype.getResources = function () {
        var _this = this;
        this.resourceService.getResources().then(function (Resources) { return _this.resources = Resources; });
    };
    ResourcesComponent.prototype.ngOnInit = function () {
        this.getResources();
    };
    ResourcesComponent.prototype.onSelect = function (resource) {
        this.selectedResource = resource;
    };
    ResourcesComponent.prototype.onBuy = function () {
        // buy action for selectedResource and user
        // create partial license
        var partialLicense = this.createPartialLicense(this.user, undefined);
        var p = new purchase_1.Purchase;
        p.label = this.selectedResource.location;
        p.partialLicense = JSON.stringify(partialLicense);
        p.resource = this.selectedResource.id;
        p.user = this.user;
        console.log(p);
        this.purchaseService.create(p);
        // create purchase in database
        // ask license on lcpserver
        console.log(this.user.alias + ' bought ' + this.selectedResource.location);
        // TODO alert user somehow...goto user details ?
        // redirect to download ?
    };
    ResourcesComponent.prototype.onLoan = function () {
        // TODO add parameters for loan action (period etc.)
        var rights = new lic.UserRights;
        rights.copy = 10;
        rights.print = 10;
        rights.start = new Date();
        rights.end = new Date(rights.start.valueOf() + 30 * 24 * 3600); // + 30 days  
        console.log(rights);
        // loan action action for selectedResource and user
        var partialLicense = this.createPartialLicense(this.user, rights);
        var p = new purchase_1.Purchase;
        p.label = this.selectedResource.location;
        p.partialLicense = JSON.stringify(partialLicense);
        p.resource = this.selectedResource.id;
        p.user = this.user;
        console.log(p);
        this.purchaseService.create(p);
        // create partial license
        // create purchase in database
        // ask license on lcpserver
        console.log(this.user.alias + ' wants to loan ' + this.selectedResource.location);
    };
    ResourcesComponent.prototype.hexToBytes = function (hex) {
        var bytes;
        var c;
        for (bytes = [], c = 0; c < hex.length; c += 1) {
            bytes.push(parseInt(hex.charAt(c), 16));
        }
        return bytes;
    };
    ResourcesComponent.prototype.createPartialLicense = function (user, rights) {
        var partialLicense = new lic.PartialLicense;
        partialLicense.provider = lic.PROVIDER;
        partialLicense.user = { id: '_' + String(user.userID), email: user.email, name: user.alias, encrypted: undefined };
        partialLicense.rights = rights;
        partialLicense.encryption = new lic.Encryption;
        partialLicense.encryption.user_key = new lic.UserKey;
        partialLicense.encryption.user_key.value = this.hexToBytes(user.password);
        partialLicense.encryption.user_key.algorithm = lic.USERKEY_ALGO;
        partialLicense.encryption.user_key.text_hint = 'Enter passphrase';
        return partialLicense;
    };
    return ResourcesComponent;
}());
__decorate([
    core_1.Input(),
    __metadata("design:type", String)
], ResourcesComponent.prototype, "id", void 0);
__decorate([
    core_1.Input(),
    __metadata("design:type", user_1.User)
], ResourcesComponent.prototype, "user", void 0);
ResourcesComponent = __decorate([
    core_1.Component({
        moduleId: module.id,
        selector: 'resources',
        templateUrl: '/app/components/resource-list.html',
        styleUrls: ['../../app/components/resource.css', '../../style.css'],
        providers: [resource_service_1.ResourceService, purchase_service_1.PurchaseService]
    }),
    __metadata("design:paramtypes", [resource_service_1.ResourceService, purchase_service_1.PurchaseService, router_1.Router])
], ResourcesComponent);
exports.ResourcesComponent = ResourcesComponent;
//# sourceMappingURL=resource-list-component.js.map