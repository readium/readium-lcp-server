import { NgModule }                 from '@angular/core';
import { RouterModule, Routes }     from '@angular/router';

import { PublicationEditComponent } from './publication-edit.component';
import { PublicationAddComponent }  from './publication-add.component';
import { PublicationListComponent } from './publication-list.component';

const publicationRoutes: Routes = [
    { path: 'publications/:id/edit', component: PublicationEditComponent },
    { path: 'publications/add', component: PublicationAddComponent },
    { path: 'publications', component: PublicationListComponent }
];

@NgModule({
  imports: [
    RouterModule.forChild(publicationRoutes)
  ],
  exports: [
    RouterModule
  ]
})

export class PublicationRoutingModule { }
