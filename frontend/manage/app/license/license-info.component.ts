import { Component } from '@angular/core';
import { License } from './license';
import { LicenseService }   from './license.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-license-info',
    templateUrl: './license-info.component.html'
})

export class LicenseInfoComponent { 
    licenses: License[];

    ngOnInit(): void {
        this.refreshInfos();
    }

    constructor(private licenseService: LicenseService) {
        
    }

    refreshInfos()
    {
        this.licenseService.get(1).then(
            infos => {
                this.licenses = infos;
            }
        );
    }
}