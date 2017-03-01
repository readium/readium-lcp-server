import { Component, Input, OnInit } from '@angular/core';
import {ActivatedRoute, Params} from '@angular/router';
import {Location} from '@angular/common';

import { User } from './user';
import { Purchase } from './purchase';
import { PurchaseService } from './purchase.service';
import {LsdService} from './lsd.service';

@Component({
    moduleId: module.id,
    selector: 'purchases',
    templateUrl: '/app/components/purchases.html',
    styleUrls: ['../../app/components/purchases.css'],
    providers: [PurchaseService, LsdService]
})

export class PurchasesComponent implements OnInit {
    @Input() user: User;
    @Input() hours: string;
    @Input() deviceID: string;
    @Input() deviceName: string;

    purchases: Purchase[];
    selectedPurchase: Purchase;

    constructor(
        private purchaseService: PurchaseService,
        private lsdService: LsdService,
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
    RegisterDevice(p: Purchase, deviceID: string, deviceName: string|undefined) {
        this.deviceID = deviceID;
        this.deviceName = deviceName;
        console.log('register license for device ' + deviceID );
        if ( p.licenseID !== '') {
            this.lsdService.registerDevice(p.licenseID, deviceID, deviceName)
            .then( status => alert('DEVICE registered!\n' + JSON.stringify(status) ) )
            .catch( reason =>  alert( 'PROBLEM: \n' +  reason._body));
        } else {
            alert('No licenseID for this purchase, please press download to create a license.');
        }
    }

    RenewLoan(p: Purchase, hours: number, deviceID: string|undefined, deviceName: string|undefined) {
        console.log('should renew license for another ' + hours + ' hours. ()' + p.label + ')');
        if ( p.licenseID !== '') {
            let t = Date.now();
            t += hours * 3600 * 1000;
            this.lsdService.renewLoan(p.licenseID, new Date(t), deviceID, deviceName)
            .then( status => alert(JSON.stringify(status) ) )
            .catch( reason =>  alert( 'RENEW PROBLEM: \n' +  reason._body));
        } else {
            alert('No licenseID for this purchase, please press download to create a license.');
        }
    }

    // contact lsd server and return the license
    ReturnLoan(p: Purchase, deviceID: string|undefined, deviceName: string|undefined) {
        if ( p.licenseID !== '') {
            this.lsdService.returnLoan(p.licenseID, deviceID, deviceName)
            .then( status => alert(JSON.stringify(status) ) )
            .catch( reason => console.log('error returning license for ' + p.label + ':' + reason) )
        } else {
            alert('No licenseID yet for this purchase! (clic download first)');
        }
    }

    // contact lsd server and CheckStatus of the license
    CheckStatus(p: Purchase) {
        if ( p.licenseID !== '') {
            this.lsdService.getStatus(p.licenseID,undefined,undefined)
            .then( status => alert(JSON.stringify(status) ) )
            .catch( reason => console.log('error checking LSD status for ' + p.label + ':' + reason) )
        } else {
            alert('No licenseID for this purchase, please press download to create a license.');
        }

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

    // get epub with license
    DownloadPublication(p: Purchase): void {
        if ( p.licenseID === undefined) {
            console.log('Get purchase and download ' + p.label );
            // existing lcp function
            // we need to recontact the static server and ask to create a new license
            window.location.href = '/users/' + p.user.userID + '/purchases/' + p.purchaseID + '/publication';
        } else {
            console.log('Re-download publication : ' + p.label + '(' + p.licenseID + ')');
            // redirect to /licenses/ p.licenseID 
            window.location.href = '/licenses/' + p.licenseID + '/publication';
        }
    }

}
