import { Component, OnInit } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute } from '@angular/router';
import { DefaultService, ModelPackage, PackageData, PackageMetadata } from 'generated';
import { LoginService } from 'loginService/login.service';

@Component({
  selector: 'app-update-page',
  templateUrl: './update-page.component.html',
  styleUrls: ['./update-page.component.scss']
})
export class UpdatePageComponent implements OnInit{
  pkg_meta: PackageMetadata
  pkg_data: PackageData
  pkg: ModelPackage
  url: string = ""

  ngOnInit(): void {
    this.route.queryParams.subscribe(params => {
      this.pkg_meta = {
        Name: params["name"],
        ID: params["id"],
        Version: params["version"]
      }
    })
  }

  constructor(private service: DefaultService, private route: ActivatedRoute, private _snackbar: MatSnackBar, private loginService: LoginService) {}

  update() {

    if (confirm("Are you sure you want to update this package?")) {
      this.pkg_data = {
        URL: this.url
      }
  
      this.pkg = {
        data: this.pkg_data,
        metadata: this.pkg_meta
      }
      this.service.packageUpdate(this.pkg, this.loginService.getToken(), this.pkg.metadata.ID).subscribe(body => {
        this._snackbar.open("Success updating package", "ok")
      }, error => {
        this._snackbar.open(error.message, "ok")
      })
    }

  }
}
