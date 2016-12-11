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
    styleUrls: ['/app/components/purchases.css'],
    providers: [PurchaseService]
})

export class PurchasesComponent implements OnInit {
    @Input() user: User;
    purchases: Purchase[];

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

}
