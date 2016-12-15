import { Component, Input, OnInit } from '@angular/core';
import {ActivatedRoute, Params} from '@angular/router';
import {Location} from '@angular/common';

import { User } from './user';
import { Purchase } from './purchase';
import { PurchaseService } from './purchase.service';
@Component({
    moduleId: module.id,
    selector: 'purchases',
    templateUrl: '/app/components/purchases.html',
    styleUrls: ['../../app/components/purchases.css'],
    providers: [PurchaseService]
})

export class PurchasesComponent implements OnInit {
    @Input() user: User;
    purchases: Purchase[];
    selectedPurchase: Purchase;

    constructor(
        private purchaseService: PurchaseService,
        private route: ActivatedRoute,
        private location: Location
    ) {}

    ngOnInit(): void {
        this.purchaseService.getPurchases(this.user)
        .then(purchases => this.purchases = purchases);
    }
    goBack(): void {
        this.location.back();
    }

    onSelect(p: Purchase): void {
        this.selectedPurchase = p;
    }

    DownloadLicense(p: Purchase): void {
        // get License !
        if ( p.licenseID === undefined) {
            console.log('Get license and download ' + p.label );
            // the license does not yet exist (some error occured ?)
            // we need to recontact the static server and ask to create a new license
            window.location.href = '/users/' + p.user.userID + '/purchases/' + p.purchaseID + '/license';
        } else {
            console.log('Re-download ' + p.label + '(' + p.licenseID + ')');
            // redirect to /licenses/ p.licenseID 
            window.location.href = '/licenses/' + p.licenseID;
        }

    }

}
