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
var resource_1 = require("./resource");
var resource_service_1 = require("./resource.service");
var user_service_1 = require("./user.service");
var ResourceComponent = (function () {
    function ResourceComponent(userService, resourceService, route, location) {
        this.userService = userService;
        this.resourceService = resourceService;
        this.route = route;
        this.location = location;
    }
    ResourceComponent.prototype.ngOnInit = function () {
        var _this = this;
        this.route.params.forEach(function (params) {
            var id = +params['id'];
            _this.userService.getUser(id)
                .then(function (user) { return _this.user = user; });
        });
    };
    ResourceComponent.prototype.goBack = function () {
        this.location.back();
    };
    ResourceComponent.prototype.save = function () {
        var _this = this;
        this.userService.update(this.user)
            .then(function () { return _this.goBack(); });
    };
    return ResourceComponent;
}());
__decorate([
    core_1.Input(),
    __metadata("design:type", user_1.User)
], ResourceComponent.prototype, "user", void 0);
__decorate([
    core_1.Input(),
    __metadata("design:type", resource_1.Resource)
], ResourceComponent.prototype, "resource", void 0);
ResourceComponent = __decorate([
    core_1.Component({
        moduleId: module.id,
        selector: 'resource',
        templateUrl: '/app/components/resource.html',
        styleUrls: ['/app/components/resource.css'],
        providers: [resource_service_1.ResourceService]
    }),
    __metadata("design:paramtypes", [user_service_1.UserService,
        resource_service_1.ResourceService,
        router_1.ActivatedRoute,
        common_1.Location])
], ResourceComponent);
exports.ResourceComponent = ResourceComponent;
//# sourceMappingURL=resource-component.js.map