<?php
interface AutoloadedContract {
    const LABEL = 'iface';
    public function name();
}

class AutoloadedCase implements AutoloadedContract {
    const LABEL = 'class';

    public function name() {
        return 'loaded';
    }
}
