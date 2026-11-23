## S1: Parcels list loads
Given: FARM-QA, /farm/30837426142555137/parcels, logged in as fe-automation (direct)
When: open the page, go to the next page of the list
Then: parcel rows render on both pages, no error snackbar, no error boundary, no red errors in the browser console

## S2: Map renders
Given: same as S1, /farm/30837426142555137/map
When: open the page, zoom in once, click a parcel on the map
Then: map tiles and parcel shapes render, the parcel detail opens, no error snackbar, no red errors in the console

## S3: Actions list and detail
Given: same as S1, /farm/30837426142555137/actions
When: open the page, open the first action
Then: the list renders, the action detail form opens with its values filled, no error snackbar, no red errors in the console

## S4: Catalogues
Given: same as S1, /farm/30837426142555137/catalogues
When: open the page, switch through its tabs
Then: every tab renders a table or an empty state, no error snackbar, no red errors in the console

## S5: Telematics
Given: same as S1, /farm/30837426142555137/telematics
When: open the page
Then: the records table or its empty state renders, date filter visible, no error snackbar, no red errors in the console
