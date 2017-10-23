import { Component, OnInit } from '@angular/core';
import { User } from './user';
import { UserService } from './user.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-user-list',
    templateUrl: 'user-list.component.html'
})

export class UserListComponent implements OnInit {
    users: User[];

    order: string;
    reverse: boolean = false;

    constructor(private userService: UserService) {
        this.users = [];
        this.order = "id";
        this.reverse = true;
    }

    refreshUsers(): void {
        this.userService.list().then(
            users => {
                this.users = users;
            }
        );
    }

    orderBy(newOrder: string)
    {
      if (newOrder == this.order)
      {
        this.reverse = !this.reverse;
      }
      else
      {
        this.reverse = false;
        this.order = newOrder
      }
    }

    ngOnInit(): void {
        this.refreshUsers();
    }

    onRemove(objId: any): void {
        this.userService.delete(objId).then(
            user => {
                this.refreshUsers();
            }
        );
    }
 }
