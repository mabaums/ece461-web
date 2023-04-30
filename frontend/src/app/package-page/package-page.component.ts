import { Component, OnInit } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute } from '@angular/router';
import { DefaultService, PackageData, PackageHistoryEntry, PackageRating } from 'generated';
import { LoginService } from 'loginService/login.service';

@Component({
  selector: 'app-package-page',
  templateUrl: './package-page.component.html',
  styleUrls: ['./package-page.component.scss']
})
export class PackagePageComponent implements OnInit {

  constructor (private route: ActivatedRoute, private service: DefaultService, private _snackbar: MatSnackBar, private loginService: LoginService) {}

  name: string = ""
  id: string = ""
  pkg_history: PackageHistoryEntry[]
  pkg_rate: PackageRating

  ngOnInit(): void {
    this.route.queryParams.subscribe( params => {
      this.name = params['name'];
      this.id = params['id'];
      this.getPackageByName()
      this.ratePackage()
    })
  }

  getPackageByName() {
    this.service.packageByNameGet(this.name, this.loginService.getToken()).subscribe(body => {
      this.pkg_history = body;
      console.log("Retrieved pkg ", this.name, " ", this.pkg_history);
    }, error => {
      this._snackbar.open(error.message, "ok")
    })
  }

  ratePackage() {
    this.service.packageRate(this.id, this.loginService.getToken()).subscribe(body => {
      this.pkg_rate = body;
      console.log("Recieved rating: ", this.pkg_rate)
    }, error => {
      // Open snackbar
      this._snackbar.open(error.message, "ok")
    })
  }
}
