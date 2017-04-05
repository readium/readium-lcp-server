import { NgModule }                 from '@angular/core';
import { BrowserModule }            from '@angular/platform-browser';
import { HttpModule }               from '@angular/http';

import { PageNotFoundComponent }    from './not-found.component';
import { AppComponent }             from './app.component';
import { AppRoutingModule }         from './app-routing.module';

import { LsdModule }                from './lsd/lsd.module';
import { SidebarModule }            from './shared/sidebar/sidebar.module';
import { HeaderModule }             from './shared/header/header.module';
import { DashboardModule }          from './dashboard/dashboard.module';
import { UserModule }               from './user/user.module';
import { PublicationModule }        from './publication/publication.module';
import { PurchaseModule }           from './purchase/purchase.module';
import { EventModule }              from './event/event.module';


@NgModule({
    imports: [
        BrowserModule,
        HttpModule,
        LsdModule,
        HeaderModule,
        SidebarModule,
        DashboardModule,
        UserModule,
        PublicationModule,
        PurchaseModule,
        AppRoutingModule,
        EventModule
    ],
    declarations: [
        AppComponent,
        PageNotFoundComponent
    ],
    bootstrap: [
        AppComponent
    ]
})

export class AppModule { }
