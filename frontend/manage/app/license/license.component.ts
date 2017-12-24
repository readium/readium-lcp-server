import { Component } from '@angular/core';
import { LicenseService } from './license.service'

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-license',
    templateUrl: 'license.component.html',
    providers: [LicenseService]
})

export class LicenseComponent { }
