import { Component, Input, OnInit } from '@angular/core';

import { Router } from '@angular/router';
import { User } from './user';
import { Purchase } from './purchase';
import { Resource } from './resource';
import  * as lic from './partialLicense';
import { ResourceService } from './resource.service';
import { PurchaseService } from './purchase.service';

@Component({
    moduleId: module.id,
    selector: 'resources',
    templateUrl: '/app/components/resource-list.html',
    styleUrls: ['../../app/components/resource.css', '../../style.css'], // from /js/app/components... 
    providers: [ResourceService, PurchaseService]
})


export class ResourcesComponent implements OnInit {
    resources: Resource[];
    selectedResource: Resource;
    @Input() id: string;
    @Input() user: User;

    constructor(private resourceService: ResourceService, private purchaseService: PurchaseService, private router: Router) { }

    
    getResources(): void {
        this.resourceService.getResources().then(Resources => this.resources = Resources);
    }

    ngOnInit(): void {
        this.getResources();
    }

    onSelect(resource: Resource): void {
        this.selectedResource = resource;
    }

    onBuy(): void {
        // buy action for selectedResource and user
        // create partial license
        let partialLicense =this.createPartialLicense(this.user, undefined);
        let p = new Purchase;
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
    }

    onLoan(): void {
        // TODO add parameters for loan action (period etc.)
        let rights = new lic.UserRights;
        rights.copy = 10;
        rights.print = 10;
        rights.start = new Date();
        rights.end = new Date( rights.start.valueOf() + 30 * 24 * 3600); // + 30 days  
        console.log(rights);
        // loan action action for selectedResource and user
        let partialLicense =this.createPartialLicense(this.user, rights);
        let p = new Purchase;
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
    }

    private hexToBytes(hex: string) {
        let bytes: any[];
        let c: number;
        for (bytes = [], c = 0; c < hex.length; c += 1) {
            bytes.push(parseInt(hex.charAt(c), 16));
        }
        return bytes;
    }

    private createPartialLicense(user: User, rights: lic.UserRights): lic.PartialLicense {
        let partialLicense = new lic.PartialLicense;
        partialLicense.provider =  lic.PROVIDER;
        partialLicense.user =  {id: '_' + String(user.userID), email: user.email, name: user.alias, encrypted: undefined };
        partialLicense.rights = rights;
        partialLicense.encryption = new lic.Encryption;
        partialLicense.encryption.user_key = new lic.UserKey;
        partialLicense.encryption.user_key.value =  this.hexToBytes( user.password);
        partialLicense.encryption.user_key.algorithm = lic.USERKEY_ALGO;
        partialLicense.encryption.user_key.text_hint = 'Enter passphrase';
        return partialLicense;
    }
}
