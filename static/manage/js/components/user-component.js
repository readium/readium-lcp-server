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
var common_1 = require('@angular/common');
var user_1 = require('./user');
var user_service_1 = require('./user.service');
var UserComponent = (function () {
    function UserComponent(userService, route, location) {
        this.userService = userService;
        this.route = route;
        this.location = location;
    }
    UserComponent.prototype.ngOnInit = function () {
        var _this = this;
        this.route.params.forEach(function (params) {
            var id = +params['id'];
            _this.userService.getUser(id)
                .then(function (user) { return _this.user = user; });
        });
    };
    UserComponent.prototype.goBack = function () {
        this.location.back();
    };
    UserComponent.prototype.save = function () {
        var _this = this;
        this.userService.update(this.user)
            .then(function () { return _this.goBack(); });
    };
    __decorate([
        core_1.Input(), 
        __metadata('design:type', user_1.User)
    ], UserComponent.prototype, "user", void 0);
    UserComponent = __decorate([
        core_1.Component({
            moduleId: module.id,
            selector: 'user',
            templateUrl: '/app/components/user.html',
            styleUrls: ['/app/components/user.css'],
            providers: [user_service_1.UserService]
        }), 
        __metadata('design:paramtypes', [user_service_1.UserService, router_1.ActivatedRoute, common_1.Location])
    ], UserComponent);
    return UserComponent;
}());
exports.UserComponent = UserComponent;
//# sourceMappingURL=user-component.js.map