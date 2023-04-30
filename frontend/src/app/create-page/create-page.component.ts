import { Component, OnInit } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';
import { DefaultService, ModelPackage, PackageData } from 'generated';
import { LoginService } from 'loginService/login.service';

@Component({
  selector: 'app-create-page',
  templateUrl: './create-page.component.html',
  styleUrls: ['./create-page.component.scss']
})
export class CreatePageComponent implements OnInit {

  ngOnInit(): void {
    if (!this.loginService.loggedIn()) {
      this.router.navigate(['/login'])
    }
  }
  
  constructor(private service: DefaultService, private _snackbar: MatSnackBar, private loginService: LoginService, private router: Router) {

  }

  name: string = ""
  version: string = ""
  URL: string = ""
  pkg_data: PackageData
  pkg: ModelPackage

  create() {
    this.pkg_data = {URL: this.URL}
    this.service.packageCreate(this.pkg_data, this.loginService.getToken()).subscribe(
      body => {
        this.pkg = body;
        this._snackbar.open("Package created", "ok");
      }, error => {
        this._snackbar.open(error.message, "ok")
      }
    )

  }
}
