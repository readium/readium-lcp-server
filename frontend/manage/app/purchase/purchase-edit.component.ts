import { Component, OnInit }        from '@angular/core';
import { ActivatedRoute, Params }   from '@angular/router';
import 'rxjs/add/operator/switchMap';

import { Purchase }                     from './purchase';
import { PurchaseService }              from './purchase.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-purchase-edit',
    templateUrl: 'purchase-edit.component.html'
})

export class PurchaseEditComponent implements OnInit {
    purchase: Purchase;

    constructor(
        private route: ActivatedRoute,
        private purchaseService: PurchaseService) {
    }

    ngOnInit(): void {
        this.route.params
            .switchMap((params: Params) => this.purchaseService.get(params['id']))
            .subscribe(purchase => {
                this.purchase = purchase
            });
    }
}
