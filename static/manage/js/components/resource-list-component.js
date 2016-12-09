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
var core_1 = require('@angular/core');
var router_1 = require('@angular/router');
var resource_service_1 = require('./resource.service');
var ResourcesComponent = (function () {
    function ResourcesComponent(ResourceService, router) {
        this.ResourceService = ResourceService;
        this.router = router;
    }
    ResourcesComponent.prototype.getResources = function () {
        var _this = this;
        this.ResourceService.getResources().then(function (Resources) { return _this.resources = Resources; });
    };
    ResourcesComponent.prototype.ngOnInit = function () {
        this.getResources();
    };
    ResourcesComponent.prototype.onSelect = function (resource) {
        this.selectedResource = resource;
    };
    __decorate([
        core_1.Input(), 
        __metadata('design:type', String)
    ], ResourcesComponent.prototype, "id", void 0);
    ResourcesComponent = __decorate([
        core_1.Component({
            moduleId: module.id,
            selector: 'resources',
            templateUrl: '/app/components/resource-list.html',
            styleUrls: ['/style.css'],
            providers: [resource_service_1.ResourceService]
        }), 
        __metadata('design:paramtypes', [resource_service_1.ResourceService, router_1.Router])
    ], ResourcesComponent);
    return ResourcesComponent;
}());
exports.ResourcesComponent = ResourcesComponent;
//# sourceMappingURL=resource-list-component.js.map