import { Component, OnInit }        from '@angular/core';
import { ActivatedRoute, Params }   from '@angular/router';
import { Observable, Subscription } from 'rxjs/Rx';
import 'rxjs/add/operator/switchMap';

import { Purchase }                 from './purchase';
import { PurchaseService }          from './purchase.service';
import { LsdService }               from '../lsd/lsd.service';
import { LicenseStatus }            from '../lsd/license-status';
import * as moment                  from 'moment';

declare var Config: any;

@Component({
    moduleId: module.id,
    selector: 'lcp-purchase-status',
    templateUrl: 'purchase-status.component.html'
})

export class PurchaseStatusComponent implements OnInit {
    purchase: Purchase;
    licenseStatus: LicenseStatus;
    revokeMessage: string = "";

    constructor(
        private route: ActivatedRoute,
        private purchaseService: PurchaseService,
        private lsdService: LsdService) {
    }

    ngOnInit(): void {
        this.refreshPurchase();
    }

    refreshPurchase(){
        this.route.params
            .switchMap((params: Params) => this.purchaseService.get(params['id']))
            .subscribe(purchase => {
                this.purchase = purchase;
                if (purchase.licenseUuid){
                    this.lsdService.get(purchase.licenseUuid).then(
                        licenseStatus => {
                            this.licenseStatus = licenseStatus;
                        });
                }
            });
    }

    formatDate(date: string): string {
        return moment(date).format('YYYY-MM-DD HH:mm');
    }

    onDownload_LSD(purchase: Purchase): void {

        // The URL does not resolve to a content-disposition+filename like "ebook_title.lsd"
        // If this were the case, most web browsers would normally just download the linked file.
        // Instead, with some browsers the file is displayed (the current page context is overwritten)
        let url = this.buildLsdDownloadUrl(purchase);
        
        //document.location.href = url;
        window.open(url, "_blank");
    }

     buildLsdDownloadUrl(purchase: Purchase): string {
        return Config.lsd.url + '/licenses/' + purchase.licenseUuid + '/status';
    }

    onDownload_LCPL(purchase: Purchase): void {
        // Wait 5 seconds before refreshing purchases
        let downloadTimer = Observable.timer(5000);
        let downloadSubscriber = downloadTimer.subscribe(
            (t: any) => {
                this.refreshPurchase();
                downloadSubscriber.unsubscribe();
            }
        );

        // The URL resolves to a content-disposition+filename like "ebook_title.lcpl"
        // Most web browsers should normally just download the linked file, not display it.
        let url = this.buildLcplDownloadUrl(purchase);

        window.open(url, "_blank");
    }

    buildLcplDownloadUrl(purchase: Purchase): string {
        return Config.frontend.url + '/api/v1/purchases/' + purchase.id + '/license';
    }

    onReturn(purchase: Purchase): void {
        purchase.status = 'to-be-returned';
        this.purchaseService.update(purchase).then(
            purchase => {
                this.refreshPurchase();
            }
        );
    }

    onRevoke(purchase: Purchase): void {
        this.purchaseService.revoke("", purchase.licenseUuid).then(
            status => {
                this.refreshPurchase();
                var success:boolean = false;
                if (status == 200) {
                    this.revokeMessage = "The license has been revoked";
                    success = true;
                } else if (status == 400){
                    this.revokeMessage = '400 The new status is not compatible with current status';
                }
                else if (status ==  401){
                    this.revokeMessage = '401 License not Found';
                }
                else if (status == 404){
                    this.revokeMessage = '404 License not Found';
                }
                else if (status >= 500){
                    this.revokeMessage = 'An internal error appear';
                }
                else{
                    this.revokeMessage = 'An internal error appear';
                }
                this.showSnackBar(success);
            }
        );
    }

    showSnackBar(success: boolean) {
        var x = $("#snackbar");
        var xClass = "show";
        if (success) xClass = "show success";
        x.attr("class",xClass);
        setTimeout(function(){ x.attr("class",""); }, 3000);
    }
    
}
