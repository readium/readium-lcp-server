import { Component, Input, OnInit }     from '@angular/core';
import { Router }                       from '@angular/router';
import {
    FormGroup,
    FormControl,
    Validators,
    FormBuilder }                       from '@angular/forms';

import { User }                         from './user';
import { UserService }                  from './user.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-user-form',
    templateUrl: 'user-form.component.html'
})

export class UserFormComponent implements OnInit {
    @Input()
    user: User;

    edit: boolean = false;
    submitButtonLabel: string = "Add";
    form: FormGroup;

    private submitted = false;

    constructor(
        private fb: FormBuilder,
        private router: Router,
        private userService: UserService) {
    }

    ngOnInit(): void {
        if (this.user == null) {
            this.user = new User();
            this.submitButtonLabel = "Add";
            this.form = this.fb.group({
                "name": ["", Validators.required],
                "email": ["", Validators.required],
                "password": ["", Validators.required],
                "hint": ["", Validators.required]
            });
        } else {
            this.edit = true;
            this.submitButtonLabel = "Save";
            this.form = this.fb.group({
                "name": [this.user.name, Validators.required],
                "email": [this.user.email, Validators.required],
                "password": "",
                "hint": [this.user.hint, Validators.required]
            });
        }
    }

    gotoList() {
        this.router.navigate(['/users']);
    }

    onCancel() {
        this.gotoList();
    }

    onSubmit() {
        this.bindForm();

        if (this.edit) {
            this.userService.update(
                this.user
            ).then(
                user => {
                    this.gotoList();
                }
            );
        } else {
            this.userService.add(this.user).then(
                user => {
                    this.gotoList();
                }
            );
        }

        this.submitted = true;
    }

    // Bind form to user
    bindForm(): void {
        this.user.name = this.form.value['name'];
        this.user.email = this.form.value['email'];
        this.user.hint = this.form.value['hint'];

        let newPassword: string = this.form.value['password'];
        newPassword = newPassword.trim();

        if (newPassword.length > 0) {
            this.user.clearPassword = newPassword;
        }
    }
}
