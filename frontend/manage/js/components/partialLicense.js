"use strict";
var __extends = (this && this.__extends) || function (d, b) {
    for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p];
    function __() { this.constructor = d; }
    d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
};
exports.PROFILE = 'http://readium.org/lcp/profile-1.0';
exports.USERKEY_ALGO = 'http://www.w3.org/2001/04/xmlenc#sha256';
exports.PROVIDER = 'http://edrlab.org';
var Key = (function () {
    function Key() {
    }
    return Key;
}());
exports.Key = Key;
var ContentKey = (function (_super) {
    __extends(ContentKey, _super);
    function ContentKey() {
        return _super.apply(this, arguments) || this;
    }
    return ContentKey;
}(Key));
exports.ContentKey = ContentKey;
var UserKey = (function (_super) {
    __extends(UserKey, _super);
    function UserKey() {
        return _super.apply(this, arguments) || this;
    }
    return UserKey;
}(Key));
exports.UserKey = UserKey;
var Encryption = (function () {
    function Encryption() {
    }
    return Encryption;
}());
exports.Encryption = Encryption;
var Link = (function () {
    function Link() {
    }
    return Link;
}());
exports.Link = Link;
;
var UserRights = (function () {
    function UserRights() {
    }
    return UserRights;
}());
exports.UserRights = UserRights;
var UserInfo = (function () {
    function UserInfo() {
    }
    return UserInfo;
}());
exports.UserInfo = UserInfo;
var PartialLicense = (function () {
    function PartialLicense() {
    }
    return PartialLicense;
}());
exports.PartialLicense = PartialLicense;
var PartialLicenseJSON = (function (_super) {
    __extends(PartialLicenseJSON, _super);
    function PartialLicenseJSON() {
        return _super.apply(this, arguments) || this;
    }
    return PartialLicenseJSON;
}(PartialLicense));
exports.PartialLicenseJSON = PartialLicenseJSON;
//# sourceMappingURL=partialLicense.js.map