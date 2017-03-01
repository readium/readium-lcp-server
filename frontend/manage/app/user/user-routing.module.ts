import { NgModule }             from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { UserListComponent }    from './user-list.component';
import { UserAddComponent }     from './user-add.component';
import { UserEditComponent }    from './user-edit.component';

const userRoutes: Routes = [
    { path: 'users/:id/edit', component: UserEditComponent },
    { path: 'users/add', component: UserAddComponent },
    { path: 'users', component: UserListComponent }
];

@NgModule({
  imports: [
    RouterModule.forChild(userRoutes)
  ],
  exports: [
    RouterModule
  ]
})

export class UserRoutingModule { }
