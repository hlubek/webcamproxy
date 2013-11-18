// Instant Webcam (c) Dominic Szablewski - PhobosLab.org

(function(window){

var canvas = document.getElementById('videoCanvas');
var notice = document.getElementById('notice');
var noticeText = document.getElementById('noticeText');

var player = null;
var client = null;
var address = 'ws://' + window.location.host + '/ws';
var reconnectInProgress = false;


// ------------------------------------------------------
// Initial connecting and retry

var connect = function() {
	reconnectInProgress = false;
	
	if( client && client.readyState == client.OPEN ) { return; }
	
	if( player ) { player.stop(); player = null; }
	if( client ) { client.close(); client = null; }
	
	client = new WebSocket( address );
	player = new jsmpeg(client, {canvas:canvas});
	
	// Attempt to reconnect when closing, on error or if the connection
	// isn't open after some time.
	client.addEventListener('close', function(){
		recorderLostConnection();
		attemptReconnect();
	}, false);
	client.addEventListener('error', attemptReconnect, false);
	setTimeout(function(){
		if( !client || client.readyState != client.OPEN ) {
			attemptReconnect();
		}
	}, 2000);
	
	// Hide notice when connected
	client.addEventListener('open', function(){ setNotice(null); }, false);
};

var setNotice = function( msg ) {
	if( !msg ) {
		notice.style.display = 'none';
	}
	else {
		notice.style.display = 'block';
		noticeText.innerHTML = msg;
	}
};

var attemptReconnect = function(event) {
	
	if( reconnectInProgress ) { return; }
	
	setTimeout(connect, 500);
	reconnectInProgress = true;
	setNotice('Lost connection');
};


// ------------------------------------------------------
// Recording

var recordButton = document.getElementById('record');
var recordDot = document.getElementById('recordDot');
var recordNotice = document.getElementById('recordNotice');
var recordStats = document.getElementById('recordStats');
var recordLinkBox = document.getElementById('recordLinkBox');
var recordLink = document.getElementById('recordLink');
recordButton.className = 'available';

var recordingStatsInterval = 0;
var recordingLastURL = null;
recordButton.onclick = function(ev) {
	ev.preventDefault();
	ev.stopPropagation();
	if( !player.canRecord() ) { return false; }
	
	if( !canRecord() ) {
		document.getElementById('recordDisabled').style.display = 'inline';
		return false;
	}
	
	if( recordButton.className == 'available' ) {
		recordButton.className = 'waiting';
		startRecording();
	}
	else if( recordButton.className == 'recording' ) {
		recordButton.className = 'available';
		stopRecordingAndDownload();
	}
	return false;
};

var canRecord = function() {
	return (window.URL && window.URL.createObjectURL);
};

var startRecording = function() {
	setRecordingState(true);
	recordLinkBox.style.display = 'none';
	
	if( recordingLastURL ) {
		if( window.URL && window.URL.revokeObjectURL ) {
			window.URL.revokeObjectURL(recordingLastURL);
		}
		else if( window.webkitURL && window.webkitURL.revokeObjectURL ) {
			window.webkitURL.revokeObjectURL(recordingLastURL);
		}
		recordingLastURL = null;
	}
	recordLink.href = '#';
	player.startRecording(recordingDidStart);
	return false;
};

var recordingDidStart = function(player) {
	recordNotice.innerHTML = 'Recording';
	recordButton.className = 'recording';
	recordStats.style.display = 'inline';
	recordingStatsInterval = setInterval(recordStatsUpdate, 33);
};

var recordStatsUpdate = function() {
	var size = (player.recordedSize/1024/1024).toFixed(2);
	recordStats.innerHTML = '(' + size +'mb)';
};

var stopRecordingAndDownload = function() {
	recordStats.style.display = 'none';
	clearInterval(recordingStatsInterval);
	setRecordingState(false);

	var today = new Date();
	var dd = today.getDate();
		dd = (dd < 10 ? '0' : '') + dd;
	var mm = today.getMonth() + 1;
		mm = (mm < 10 ? '0' : '') + mm;
	var yyyy = today.getFullYear();
	var hh = today.getHours();
		hh = (hh < 10 ? '0' : '') + hh;
	var ii = today.getMinutes();
		ii = (ii < 10 ? '0' : '') + ii;
	
	var fileName = 'Webcam-'+yyyy+'-'+mm+'-'+dd+'-'+hh+'-'+ii+'.mpg';
	var size = (player.recordedSize/1024/1024).toFixed(2);
	
	
	recordLink.innerHTML = fileName + ' (' + size +'mb)';
	recordLink.download = fileName;
	
	var blob = player.stopRecording();
	recordingLastURL = window.URL.createObjectURL(blob);
	recordLink.href = recordingLastURL;
	recordLinkBox.style.display = 'inline';
};

var recorderLostConnection = function() {
	if( recordButton.className == 'recording' ) {
		recordButton.className = 'available';
		stopRecordingAndDownload();
	}
};

var setRecordingState = function(enabled) {
	recordDot.innerHTML = enabled ? '&#x25cf;' : '&#x25cb;';
	recordNotice.innerHTML = enabled ? 'Recording' : 'Record';
};


// ------------------------------------------------------
// Init!

if( navigator.userAgent.match(/iPhone|iPod|iPad|iOS/i) ) {
	// Don't show recording button on iOS devices. Desktop browsers unable
	// of recording, will see a message when the record button is clicked.
	document.getElementById('record').style.display = 'none';
}

canvas.addEventListener('click', function(){
	canvas.className = (canvas.className == 'full' ? 'unscaled' : 'full');
	return false;
},false);

if( !window.WebSocket ) {
	setNotice("Your Browser doesn't Support WebSockets. Please use Firefox, Chrome, Safari or IE10");
}
else {
	setNotice('Connecting');
	connect();
}

})(window);
