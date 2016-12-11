import { NgModule }      from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule }   from '@angular/forms';

import { AppComponent }  from './components/app.component';
import { UsersComponent }  from './components/user-list-component';
import { UserComponent }  from './components/user-component';
import { PurchasesComponent }  from './components/purchase-list-component';

import { ResourcesComponent } from './components/resource-list-component';
import { HttpModule } from '@angular/http';
import { AppRoutingModule } from './app-routing.module';

@NgModule({
  imports:      [ BrowserModule, FormsModule, HttpModule, AppRoutingModule ],
  declarations: [ AppComponent, UsersComponent, UserComponent, ResourcesComponent, PurchasesComponent ],
  bootstrap:    [ AppComponent ]
})
export class AppModule { }
