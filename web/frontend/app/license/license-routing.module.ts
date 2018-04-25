import { NgModule }             from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { LicenseComponent }       from './license.component';

const licenseRoutes: Routes = [
    { path: 'licenses', component: LicenseComponent }
];

@NgModule({
  imports: [
    RouterModule.forChild(licenseRoutes)
  ],
  exports: [
    RouterModule
  ]
})

export class LicenseRoutingModule { }
