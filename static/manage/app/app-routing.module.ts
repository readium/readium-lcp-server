import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';
import { UsersComponent } from './components/user-list-component';
import { UserComponent } from './components/user-component';


const routes: Routes = [
    { path: '', redirectTo: 'userlist', pathMatch: 'full' },
    { path: 'userlist' , component: UsersComponent },
    { path: 'userdetail/:id', component: UserComponent }
];

@NgModule({
    imports: [ RouterModule.forRoot(routes)],
    exports: [ RouterModule ]
})
export class AppRoutingModule { }
