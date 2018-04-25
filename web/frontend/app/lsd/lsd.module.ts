import { NgModule }                 from '@angular/core';
import { CommonModule }             from '@angular/common';
import { RouterModule }             from '@angular/router';
import {
    FormsModule,
    ReactiveFormsModule }           from '@angular/forms';

import { Ng2DatetimePickerModule }  from 'ng2-datetime-picker';

import { LsdService }               from './lsd.service';

@NgModule({
    imports: [
        CommonModule
    ],
    declarations: [],
    providers: [
        LsdService
    ]
})

export class LsdModule { }
