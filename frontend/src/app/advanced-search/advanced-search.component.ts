import { Component, OnInit, ViewChild, AfterViewInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { PackageResultsComponent } from '../package-results/package-results.component';
@Component({
  selector: 'app-advanced-search',
  templateUrl: './advanced-search.component.html',
  styleUrls: ['./advanced-search.component.scss'],
})
export class AdvancedSearchComponent implements OnInit {

  name: string = ""
  version: string = ""

  @ViewChild(PackageResultsComponent) child:PackageResultsComponent

  constructor (private route: ActivatedRoute, private router: Router) {

  }

  ngOnInit(): void {
    this.route.queryParams.subscribe( params=> {
      this.name = params['name'];
      this.version = params['version'];
      console.log("Searching for ", this.name)
      if (this.child) {
        this.search()
      }
    })
  }

  ngAferViewInit(): void {

  }

  search() {
    this.child.name = this.name
    this.child.version = this.version
    this.router.navigate(['/search/advanced'], {queryParams:{name: this.name, version: this.version} });
    this.child.searchByNameVersion(true)
  }
}
