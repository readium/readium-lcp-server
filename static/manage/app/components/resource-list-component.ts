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
        //TODO alert user somehow...goto user details ?
    }

    onLoan(): void {
        // should add parameters for loan action (period etc.)
        /* "rights": {\n\
        "print": 10,\n\
        "copy": 2048,\n\
        "start": "2016-09-01T01:08:15+01:00",\n\
        "end": "2017-11-25T01:08:15+01:00"\n\
        }\n\*/
        // loan action action for selectedResource and user
        // create partial license
        // create purchase in database
        // ask license on lcpserver
        console.log(this.user.alias + ' wants to loan ' + this.selectedResource.location);
    }

    private createPartialLicense(user: User, rights: lic.UserRights): lic.PartialLicense {
        let partialLicense = new lic.PartialLicense;
        partialLicense.provider =  lic.PROVIDER;
        partialLicense.user =  {id: user.userID, email: user.email, name: user.alias, encrypted: undefined };
        partialLicense.rights = rights;
        partialLicense.encryption = new lic.Encryption;
        partialLicense.encryption.user_key = new lic.UserKey;
        partialLicense.encryption.user_key.clear_value = user.password;
        partialLicense.encryption.user_key.algorithm = lic.USERKEY_ALGO;
        partialLicense.encryption.user_key.text_hint = 'Enter passphrase';
        return partialLicense;
    }
}
