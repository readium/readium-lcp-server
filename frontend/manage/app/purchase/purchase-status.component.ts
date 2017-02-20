import { Component, OnInit }        from '@angular/core';
import { ActivatedRoute, Params }   from '@angular/router';
import 'rxjs/add/operator/switchMap';

import { Purchase }                 from './purchase';
import { PurchaseService }          from './purchase.service';
import { LsdService }               from '../lsd/lsd.service';
import { LicenseStatus }            from '../lsd/license-status';

@Component({
    moduleId: module.id,
    selector: 'lcp-purchase-status',
    templateUrl: 'purchase-status.component.html'
})

export class PurchaseStatusComponent implements OnInit {
    purchase: Purchase;
    licenseStatus: LicenseStatus;

    constructor(
        private route: ActivatedRoute,
        private purchaseService: PurchaseService,
        private lsdService: LsdService) {
    }

    ngOnInit(): void {
        this.route.params
            .switchMap((params: Params) => this.purchaseService.get(params['id']))
            .subscribe(purchase => {
                this.purchase = purchase;
                this.lsdService.get(purchase.licenseUuid).then(
                    licenseStatus => {
                        this.licenseStatus = licenseStatus;
                    });
            });
    }
}
