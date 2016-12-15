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
var user_service_1 = require("./user.service");
var UsersComponent = (function () {
    function UsersComponent(UserService, router) {
        this.UserService = UserService;
        this.router = router;
    }
    UsersComponent.prototype.getUsers = function () {
        var _this = this;
        this.UserService.getUsers().then(function (Users) { return _this.users = Users; });
    };
    UsersComponent.prototype.add = function (alias, email, password) {
        var _this = this;
        email = email.trim();
        if (!email) {
            return;
        }
        ;
        this.UserService.create(alias, email, password)
            .then(function (User) {
            _this.getUsers(); // refresh user list 
        });
    };
    UsersComponent.prototype.delete = function (user) {
        var _this = this;
        console.log('delete user ' + user.alias + ' ' + user.email + ' ' + user.userID);
        this.UserService
            .delete(user.userID)
            .then(function () {
            _this.users = _this.users.filter(function (h) { return h !== user; });
            if (_this.selectedUser === user) {
                _this.selectedUser = null;
            }
        });
    };
    UsersComponent.prototype.ngOnInit = function () {
        this.getUsers();
    };
    UsersComponent.prototype.onSelect = function (User) {
        this.selectedUser = User;
    };
    UsersComponent.prototype.gotoDetail = function () {
        this.router.navigate(['/userdetail', this.selectedUser.userID]);
    };
    return UsersComponent;
}());
__decorate([
    core_1.Input(),
    __metadata("design:type", String)
], UsersComponent.prototype, "alias", void 0);
__decorate([
    core_1.Input(),
    __metadata("design:type", String)
], UsersComponent.prototype, "email", void 0);
__decorate([
    core_1.Input(),
    __metadata("design:type", String)
], UsersComponent.prototype, "password", void 0);
UsersComponent = __decorate([
    core_1.Component({
        moduleId: module.id,
        selector: 'users',
        templateUrl: '/app/components/user-list.html',
        styleUrls: ['../../app/components/user.css'],
        providers: [user_service_1.UserService]
    }),
    __metadata("design:paramtypes", [user_service_1.UserService, router_1.Router])
], UsersComponent);
exports.UsersComponent = UsersComponent;
//# sourceMappingURL=user-list-component.js.map