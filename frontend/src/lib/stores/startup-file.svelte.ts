/**
 * Tracks whether desktop startup file delivery has been checked.
 * Empty launches may seed the editor with a default example only after this
 * check completes, otherwise cold file-open launches briefly render the
 * default example and then restart preview for the real file.
 */
let _startupFileCheckPending = $state(true);

export const startupFileStore = {
	get checkPending() {
		return _startupFileCheckPending;
	},

	markChecked() {
		_startupFileCheckPending = false;
	},
};
